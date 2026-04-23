package router

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// CapabilityQueryMessage represents the initial QUERY from Plugin A (Step 1).
type CapabilityQueryMessage struct {
	QueryID          string                 `json:"query_id"`
	TargetCapability string                 `json:"target_capability"`
	RequesterPluginID string                `json:"requester_plugin_id"`
	Params           map[string]interface{} `json:"params"`
	AwaitResponse    bool                   `json:"await_response"`
	CorrelationID    string                 `json:"correlation_id"`
}

// CapabilityQueryDispatch represents the QUERY_DISPATCH from Core to Plugin B (Step 2).
type CapabilityQueryDispatch struct {
	QueryID           string                 `json:"query_id"`
	RequesterPluginID string                 `json:"requester_plugin_id"`
	TargetCapability  string                 `json:"target_capability"`
	Params            map[string]interface{} `json:"params"`
	CorrelationID     string                 `json:"correlation_id"`
}

// CapabilityAck represents the ACK from Plugin B to Core (Step 3).
type CapabilityAck struct {
	QueryID  string `json:"query_id"`
	PluginID string `json:"plugin_id"`
}

// CapabilityAckForwarded represents the ACK_FORWARDED from Core to Plugin A (Step 4).
type CapabilityAckForwarded struct {
	QueryID string `json:"query_id"`
}

// CapabilityResponse represents the RESPONSE from Plugin B to Core (Step 5).
type CapabilityResponse struct {
	QueryID  string                 `json:"query_id"`
	PluginID string                 `json:"plugin_id"`
	Result   map[string]interface{} `json:"result"`
	Error    string                 `json:"error,omitempty"`
}

// CapabilityResponseDelivered represents the RESPONSE_DELIVERED from Core to Plugin A (Step 6).
type CapabilityResponseDelivered struct {
	QueryID       string                 `json:"query_id"`
	Result        map[string]interface{} `json:"result"`
	CorrelationID string                 `json:"correlation_id"`
	Error         string                 `json:"error,omitempty"`
}

// CapabilityReceiptAck represents the RECEIPT_ACK from Plugin A to Core (Step 7).
type CapabilityReceiptAck struct {
	QueryID  string `json:"query_id"`
	PluginID string `json:"plugin_id"`
}

// CapabilityRouter tracks inflight double handshake queries.
type CapabilityRouter struct {
	mu       sync.RWMutex
	bus      *eventbus.Bus
	registry *registry.Registry
	inflight map[string]*inflightQuery
}

type queryState string

const (
	QueryStateDispatched   queryState = "DISPATCHED"
	QueryStateAcked        queryState = "ACKED"
	QueryStateResponded    queryState = "RESPONDED"
	QueryStateCompleted    queryState = "COMPLETED"
	QueryStateFailed       queryState = "FAILED"
	QueryStateTimeout      queryState = "TIMEOUT"
)

type inflightQuery struct {
	queryID           string
	requesterPluginID string
	targetPluginID    string
	correlationID     string
	state             queryState
	dispatchedAt      time.Time
}

const CapACKTimeout = 500 * time.Millisecond
const CapProcessTimeout = 30 * time.Second

// NewCapabilityRouter creates a new router for double handshake capability queries.
func NewCapabilityRouter(bus *eventbus.Bus, reg *registry.Registry) *CapabilityRouter {
	cr := &CapabilityRouter{
		bus:      bus,
		registry: reg,
		inflight: make(map[string]*inflightQuery),
	}

	if bus != nil && bus.Conn() != nil {
		_, _ = bus.Subscribe("kognis.capability.query", cr.handleQuery)
		_, _ = bus.Subscribe("kognis.capability.ack", cr.handleAck)
		_, _ = bus.Subscribe("kognis.capability.response", cr.handleResponse)
		_, _ = bus.Subscribe("kognis.capability.receipt_ack", cr.handleReceiptAck)
	}

	return cr
}

// handleQuery processes Step 1 (QUERY) and performs Step 2 (QUERY_DISPATCH).
func (cr *CapabilityRouter) handleQuery(msg *nats.Msg) {
	var q CapabilityQueryMessage
	if err := json.Unmarshal(msg.Data, &q); err != nil {
		log.Printf("caprouter: invalid QUERY: %v", err)
		return
	}

	providers := cr.registry.FindByCapability(q.TargetCapability)
	if len(providers) == 0 {
		log.Printf("caprouter: no providers for capability %s", q.TargetCapability)
		cr.sendErrorResponse(q.RequesterPluginID, q.QueryID, q.CorrelationID, "no_providers")
		return
	}

	targetPluginID := providers[0].ID

	cr.mu.Lock()
	cr.inflight[q.QueryID] = &inflightQuery{
		queryID:           q.QueryID,
		requesterPluginID: q.RequesterPluginID,
		targetPluginID:    targetPluginID,
		correlationID:     q.CorrelationID,
		state:             QueryStateDispatched,
		dispatchedAt:      time.Now(),
	}
	cr.mu.Unlock()

	dispatchMsg := CapabilityQueryDispatch{
		QueryID:           q.QueryID,
		RequesterPluginID: q.RequesterPluginID,
		TargetCapability:  q.TargetCapability,
		Params:            q.Params,
		CorrelationID:     q.CorrelationID,
	}

	subject := fmt.Sprintf("kognis.capability.dispatch.%s", targetPluginID)
	data, _ := json.Marshal(dispatchMsg)
	_ = cr.bus.Publish(subject, data)

	log.Printf("caprouter: dispatched query %s to %s", q.QueryID, targetPluginID)
}

// handleAck processes Step 3 (ACK) and performs Step 4 (ACK_FORWARDED).
func (cr *CapabilityRouter) handleAck(msg *nats.Msg) {
	var ack CapabilityAck
	if err := json.Unmarshal(msg.Data, &ack); err != nil {
		return
	}

	cr.mu.Lock()
	iq, ok := cr.inflight[ack.QueryID]
	if !ok {
		cr.mu.Unlock()
		return
	}
	iq.state = QueryStateAcked
	requesterID := iq.requesterPluginID
	cr.mu.Unlock()

	fwd := CapabilityAckForwarded{QueryID: ack.QueryID}
	data, _ := json.Marshal(fwd)
	subject := fmt.Sprintf("kognis.capability.ack_forwarded.%s", requesterID)
	_ = cr.bus.Publish(subject, data)

	log.Printf("caprouter: forwarded ACK for query %s to %s", ack.QueryID, requesterID)
}

// handleResponse processes Step 5 (RESPONSE) and performs Step 6 (RESPONSE_DELIVERED).
func (cr *CapabilityRouter) handleResponse(msg *nats.Msg) {
	var resp CapabilityResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}

	cr.mu.Lock()
	iq, ok := cr.inflight[resp.QueryID]
	if !ok {
		cr.mu.Unlock()
		return
	}
	iq.state = QueryStateResponded
	requesterID := iq.requesterPluginID
	corrID := iq.correlationID
	cr.mu.Unlock()

	deliv := CapabilityResponseDelivered{
		QueryID:       resp.QueryID,
		Result:        resp.Result,
		CorrelationID: corrID,
		Error:         resp.Error,
	}
	data, _ := json.Marshal(deliv)
	subject := fmt.Sprintf("kognis.capability.response_delivered.%s", requesterID)
	_ = cr.bus.Publish(subject, data)

	log.Printf("caprouter: delivered RESPONSE for query %s to %s", resp.QueryID, requesterID)
}

// handleReceiptAck processes Step 7 (RECEIPT_ACK).
func (cr *CapabilityRouter) handleReceiptAck(msg *nats.Msg) {
	var rack CapabilityReceiptAck
	if err := json.Unmarshal(msg.Data, &rack); err != nil {
		return
	}

	cr.mu.Lock()
	iq, ok := cr.inflight[rack.QueryID]
	if ok {
		iq.state = QueryStateCompleted
		delete(cr.inflight, rack.QueryID)
		log.Printf("caprouter: completed query %s", rack.QueryID)
	}
	cr.mu.Unlock()
}

// CheckTimeouts checks for timeout queries.
func (cr *CapabilityRouter) CheckTimeouts() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	now := time.Now()
	for _, iq := range cr.inflight {
		if iq.state == QueryStateDispatched {
			if now.Sub(iq.dispatchedAt) >= CapACKTimeout {
				iq.state = QueryStateTimeout
				log.Printf("caprouter: query %s ACK timeout", iq.queryID)
				cr.sendErrorResponse(iq.requesterPluginID, iq.queryID, iq.correlationID, "timeout_ack")
			}
		} else if iq.state == QueryStateAcked {
			if now.Sub(iq.dispatchedAt) >= CapProcessTimeout {
				iq.state = QueryStateTimeout
				log.Printf("caprouter: query %s process timeout", iq.queryID)
				cr.sendErrorResponse(iq.requesterPluginID, iq.queryID, iq.correlationID, "timeout_process")
			}
		}
	}
}

// sendErrorResponse sends an error RESPONSE_DELIVERED directly to Plugin A.
func (cr *CapabilityRouter) sendErrorResponse(requesterID, queryID, corrID, errCode string) {
	deliv := CapabilityResponseDelivered{
		QueryID:       queryID,
		CorrelationID: corrID,
		Error:         errCode,
	}
	data, _ := json.Marshal(deliv)
	subject := fmt.Sprintf("kognis.capability.response_delivered.%s", requesterID)
	_ = cr.bus.Publish(subject, data)
}
