package router

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/envelope"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// InflightStatus tracks the dispatch state of a message per SPEC 04 Section 4.4.
type InflightStatus string

const (
	StatusAwaitingACK InflightStatus = "AWAITING_ACK"
	StatusProcessing  InflightStatus = "PROCESSING"
	StatusComplete    InflightStatus = "COMPLETE"
	StatusFailed      InflightStatus = "FAILED"
	StatusTimeout     InflightStatus = "TIMEOUT"
)

// ACKTimeout is the 500ms window for a plugin to ACK a dispatched message
// per SPEC 04 Section 4.4.
const ACKTimeout = 500 * time.Millisecond

// InflightMessage tracks a message that has been dispatched to a plugin
// but has not yet completed. Per SPEC 04 Section 4.4.
type InflightMessage struct {
	MsgID        string
	Envelope     *envelope.Envelope
	Pipeline     string
	Slot         string
	PluginID     string
	DispatchedAt time.Time
	Deadline     time.Time
	Status       InflightStatus
	// providerIndex tracks which provider in the slot's provider list
	// is currently handling this message (for retry on failure).
	providerIndex int
	// providers is the ordered list of plugin IDs capable of handling this slot.
	providers []string
}

// ackPayload is the JSON body sent by a plugin when it ACKs a message.
type ackPayload struct {
	MsgID               string `json:"msg_id"`
	PluginID            string `json:"plugin_id"`
	EstimatedProcessingMS int  `json:"estimated_processing_ms"`
}

// completePayload is the JSON body for a message completion notification.
type completePayload struct {
	MsgID  string `json:"msg_id"`
	Result []byte `json:"result"`
}

// failedPayload is the JSON body for a message failure notification.
type failedPayload struct {
	MsgID      string `json:"msg_id"`
	PluginID   string `json:"plugin_id"`
	ErrorCode  string `json:"error_code"`
	RetrySafe  bool   `json:"retry_safe"`
}

// MessageRouter routes envelopes through pipelines with slot-by-slot dispatch
// and inflight tracking per SPEC 04 Section 4.4.
type MessageRouter struct {
	router   *Router
	registry *registry.Registry
	bus      *eventbus.Bus
	inflight map[string]*InflightMessage // msg_id -> tracking
	mu       sync.RWMutex
}

// NewMessageRouter creates a new MessageRouter bound to the given Router,
// Registry, and EventBus.
func NewMessageRouter(rtr *Router, reg *registry.Registry, bus *eventbus.Bus) *MessageRouter {
	mr := &MessageRouter{
		router:   rtr,
		registry: reg,
		bus:      bus,
		inflight: make(map[string]*InflightMessage),
	}

	// Subscribe to ACK, COMPLETE, and FAILED subjects for inflight tracking.
	if bus != nil && bus.Conn() != nil {
		if _, err := bus.Subscribe("kognis.dispatch.ack", mr.handleACKMsg); err != nil {
			log.Printf("msgrouter: failed to subscribe to ack: %v", err)
		}
		if _, err := bus.Subscribe("kognis.dispatch.complete", mr.handleCompleteMsg); err != nil {
			log.Printf("msgrouter: failed to subscribe to complete: %v", err)
		}
		if _, err := bus.Subscribe("kognis.dispatch.failed", mr.handleFailedMsg); err != nil {
			log.Printf("msgrouter: failed to subscribe to failed: %v", err)
		}
	}

	return mr
}

// handleACKMsg is the NATS handler for ACK messages.
func (mr *MessageRouter) handleACKMsg(msg *nats.Msg) {
	var payload ackPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("msgrouter: invalid ACK payload: %v", err)
		return
	}
	if err := mr.HandleACK(payload.MsgID, payload.PluginID, payload.EstimatedProcessingMS); err != nil {
		log.Printf("msgrouter: HandleACK error: %v", err)
	}
}

// handleCompleteMsg is the NATS handler for COMPLETE messages.
func (mr *MessageRouter) handleCompleteMsg(msg *nats.Msg) {
	var payload completePayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("msgrouter: invalid COMPLETE payload: %v", err)
		return
	}
	if err := mr.HandleComplete(payload.MsgID, payload.Result); err != nil {
		log.Printf("msgrouter: HandleComplete error: %v", err)
	}
}

// handleFailedMsg is the NATS handler for FAILED messages.
func (mr *MessageRouter) handleFailedMsg(msg *nats.Msg) {
	var payload failedPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("msgrouter: invalid FAILED payload: %v", err)
		return
	}
	if err := mr.HandleFailed(payload.MsgID, payload.ErrorCode, payload.RetrySafe); err != nil {
		log.Printf("msgrouter: HandleFailed error: %v", err)
	}
}

// RouteMessage is the main routing method. It validates the envelope,
// increments the hop count, resolves the pipeline and slot, and dispatches
// to the first available provider. Per SPEC 04 Section 4.4.
func (mr *MessageRouter) RouteMessage(env *envelope.Envelope) error {
	// Step 1: Validate the envelope.
	if err := env.Validate(); err != nil {
		return fmt.Errorf("envelope validation failed: %w", err)
	}

	// Step 2: Increment hop count.
	if err := env.IncrementHop(); err != nil {
		return fmt.Errorf("hop count exceeded: %w", err)
	}

	// Step 3: Determine pipeline from envelope's Pipeline field.
	pipelineName := env.Pipeline
	if pipelineName == "" {
		return fmt.Errorf("envelope has no pipeline specified")
	}

	// Step 4: Verify pipeline exists.
	if _, ok := mr.router.GetPipeline(pipelineName); !ok {
		return fmt.Errorf("pipeline %s not found", pipelineName)
	}

	// Step 5: Find the current slot (from envelope's Slot field, or entry slot).
	slotName := env.Slot
	if slotName == "" {
		slotName = mr.findEntrySlot(pipelineName)
	}
	if slotName == "" {
		return fmt.Errorf("no entry slot found for pipeline %s", pipelineName)
	}

	// Step 6: Find providers for that slot, sorted by priority.
	pluginEntries := mr.registry.FindByPipelineSlot(pipelineName, slotName)
	if len(pluginEntries) == 0 {
		return fmt.Errorf("no providers for slot %s in pipeline %s", slotName, pipelineName)
	}

	// Sort by priority (ascending) using slot registration priority.
	sort.Slice(pluginEntries, func(i, j int) bool {
		pri := getSlotPriority(pluginEntries[i], pipelineName, slotName)
		prj := getSlotPriority(pluginEntries[j], pipelineName, slotName)
		return pri < prj
	})

	providers := make([]string, 0, len(pluginEntries))
	for _, p := range pluginEntries {
		providers = append(providers, p.ID)
	}

	// Step 7: Dispatch to the first available provider via NATS.
	pluginID := providers[0]
	subject := eventbus.SlotSubject(pipelineName, slotName)

	data, err := env.Serialize()
	if err != nil {
		return fmt.Errorf("serialize envelope: %w", err)
	}

	if err := mr.bus.Publish(subject, data); err != nil {
		return fmt.Errorf("dispatch to slot %s/%s: %w", pipelineName, slotName, err)
	}

	// Also publish on the pipeline-wide subject.
	pipeSubject := eventbus.PipelineSubject(pipelineName)
	_ = mr.bus.Publish(pipeSubject, data)

	// Step 8: Track as inflight message.
	pipeline, _ := mr.router.GetPipeline(pipelineName)
	deadline := time.Time{}
	if pipeline != nil {
		for _, slot := range pipeline.Slots {
			if slot.Name == slotName && slot.TimeoutSeconds > 0 {
				deadline = time.Now().Add(time.Duration(slot.TimeoutSeconds) * time.Second)
				break
			}
		}
	}
	if deadline.IsZero() {
		// Default deadline: 30 seconds if slot has no timeout.
		deadline = time.Now().Add(30 * time.Second)
	}

	mr.mu.Lock()
	mr.inflight[env.ID] = &InflightMessage{
		MsgID:         env.ID,
		Envelope:      env,
		Pipeline:      pipelineName,
		Slot:          slotName,
		PluginID:      pluginID,
		DispatchedAt:  time.Now(),
		Deadline:      deadline,
		Status:        StatusAwaitingACK,
		providerIndex: 0,
		providers:     providers,
	}
	mr.mu.Unlock()

	log.Printf("msgrouter: dispatched %s to %s/%s (plugin %s)", env.ID, pipelineName, slotName, pluginID)
	return nil
}

// findEntrySlot returns the first slot marked as a valid entry point.
func (mr *MessageRouter) findEntrySlot(pipelineName string) string {
	pipeline, ok := mr.router.GetPipeline(pipelineName)
	if !ok {
		return ""
	}
	for _, slot := range pipeline.Slots {
		if slot.ValidEntryPoint {
			return slot.Name
		}
	}
	return ""
}

// HandleACK updates an inflight message's status from AWAITING_ACK to PROCESSING.
// Per SPEC 04 Section 4.4, the plugin must ACK within 500ms.
func (mr *MessageRouter) HandleACK(msgID string, pluginID string, estimatedProcessingMS int) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	im, ok := mr.inflight[msgID]
	if !ok {
		return fmt.Errorf("message %s not found in inflight tracking", msgID)
	}

	if im.Status != StatusAwaitingACK {
		return fmt.Errorf("message %s is in state %s, cannot ACK", msgID, im.Status)
	}

	im.Status = StatusProcessing
	im.PluginID = pluginID

	log.Printf("msgrouter: ACK received for %s from %s (est %dms)", msgID, pluginID, estimatedProcessingMS)
	return nil
}

// HandleComplete marks a message as complete and routes it to the next slot
// in the pipeline. Per SPEC 04 Section 4.4.
func (mr *MessageRouter) HandleComplete(msgID string, result []byte) error {
	mr.mu.Lock()
	im, ok := mr.inflight[msgID]
	if !ok {
		mr.mu.Unlock()
		return fmt.Errorf("message %s not found in inflight tracking", msgID)
	}

	im.Status = StatusComplete
	// Remove the completed inflight entry BEFORE re-routing,
	// so RouteMessage can create a fresh entry for the next slot.
	delete(mr.inflight, msgID)
	mr.mu.Unlock()

	log.Printf("msgrouter: message %s completed in %s/%s", msgID, im.Pipeline, im.Slot)

	// Determine the next slot in the pipeline.
	nextSlot := mr.findNextSlot(im.Pipeline, im.Slot)
	if nextSlot == "" {
		// No more slots; message has completed the pipeline.
		log.Printf("msgrouter: message %s completed pipeline %s (no more slots)", msgID, im.Pipeline)
		return nil
	}

	// Route to the next slot.
	im.Envelope.Slot = nextSlot
	if err := mr.RouteMessage(im.Envelope); err != nil {
		return fmt.Errorf("route to next slot %s: %w", nextSlot, err)
	}

	return nil
}

// findNextSlot returns the slot that follows currentSlot in the pipeline,
// or empty string if there is no next slot.
func (mr *MessageRouter) findNextSlot(pipelineName, currentSlot string) string {
	pipeline, ok := mr.router.GetPipeline(pipelineName)
	if !ok {
		return ""
	}

	found := false
	for _, slot := range pipeline.Slots {
		if found {
			return slot.Name
		}
		if slot.Name == currentSlot {
			found = true
		}
	}
	return ""
}

// HandleFailed processes a failure for an inflight message. If retrySafe is true
// and there are more providers for the slot, it retries with the next provider.
// Otherwise it marks the message as failed. Per SPEC 04 Section 4.4.
func (mr *MessageRouter) HandleFailed(msgID string, errorCode string, retrySafe bool) error {
	mr.mu.Lock()
	im, ok := mr.inflight[msgID]
	if !ok {
		mr.mu.Unlock()
		return fmt.Errorf("message %s not found in inflight tracking", msgID)
	}

	if !retrySafe || im.providerIndex >= len(im.providers)-1 {
		// No retry possible; mark as failed.
		im.Status = StatusFailed
		mr.mu.Unlock()
		log.Printf("msgrouter: message %s FAILED (code=%s, retrySafe=%v)", msgID, errorCode, retrySafe)
		return nil
	}

	// Retry with next provider.
	im.providerIndex++
	nextPluginID := im.providers[im.providerIndex]
	im.PluginID = nextPluginID
	im.Status = StatusAwaitingACK
	im.DispatchedAt = time.Now()

	// Re-dispatch.
	subject := eventbus.SlotSubject(im.Pipeline, im.Slot)
	data, err := im.Envelope.Serialize()
	if err != nil {
		im.Status = StatusFailed
		mr.mu.Unlock()
		return fmt.Errorf("serialize envelope for retry: %w", err)
	}

	mr.mu.Unlock()

	if err := mr.bus.Publish(subject, data); err != nil {
		mr.mu.Lock()
		im.Status = StatusFailed
		mr.mu.Unlock()
		return fmt.Errorf("retry dispatch to slot %s/%s: %w", im.Pipeline, im.Slot, err)
	}

	log.Printf("msgrouter: retrying %s with provider %s (attempt %d)", msgID, nextPluginID, im.providerIndex+1)
	return nil
}

// CheckTimeouts scans for inflight messages that have exceeded their deadline
// or the ACK timeout window and marks them as TIMEOUT. Per SPEC 04 Section 4.4.
func (mr *MessageRouter) CheckTimeouts() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	now := time.Now()
	for _, im := range mr.inflight {
		switch im.Status {
		case StatusAwaitingACK:
			// AWAITING_ACK has a 500ms window per SPEC 04 Section 4.4.
			if now.Sub(im.DispatchedAt) >= ACKTimeout {
				log.Printf("msgrouter: message %s ACK timeout (500ms exceeded)", im.MsgID)
				im.Status = StatusTimeout
			}
		case StatusProcessing:
			// Check if the slot's deadline has been exceeded.
			if !im.Deadline.IsZero() && now.After(im.Deadline) {
				log.Printf("msgrouter: message %s processing timeout (deadline exceeded)", im.MsgID)
				im.Status = StatusTimeout
			}
		}
	}
}

// GetInflightCount returns the number of messages currently tracked as inflight
// (excluding COMPLETE, FAILED, and TIMEOUT messages).
func (mr *MessageRouter) GetInflightCount() int {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	count := 0
	for _, im := range mr.inflight {
		if im.Status == StatusAwaitingACK || im.Status == StatusProcessing {
			count++
		}
	}
	return count
}

// getSlotPriority returns the priority for a plugin's slot registration.
// Lower number = higher priority. Returns 999 (lowest) if not found.
func getSlotPriority(entry *registry.PluginEntry, pipeline, slot string) int {
	for _, sr := range entry.SlotRegistrations {
		if sr.Pipeline == pipeline && sr.Slot == slot {
			return sr.Priority
		}
	}
	return 999
}