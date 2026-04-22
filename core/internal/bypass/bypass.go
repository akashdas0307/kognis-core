// Package bypass implements the emergency bypass channel per SPEC 14.
//
// The bypass channel provides a high-priority path for emergency signals that
// must circumvent the normal pipeline flow. Only authorized plugins may use
// specific bypass types, and all bypass events are logged for abuse detection
// (SPEC 14 Section 14.4).
package bypass

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// Valid bypass types per SPEC 14 Section 14.2.
const (
	BypassTypeSafetySoundDetected = "safety_sound_detected"
	BypassTypeHealthCritical      = "health_critical"
	BypassTypeCreatorEmergency    = "creator_emergency"
	BypassTypePhysicalHazard      = "physical_hazard"

	// EmergencyBypassSubject is the NATS subject for bypass events.
	EmergencyBypassSubject = "kognis.emergency.bypass"

	// EmergencyBypassRequestSubject is the NATS subject plugins publish to
	// when requesting an emergency bypass.
	EmergencyBypassRequestSubject = "kognis.emergency.bypass.request"
)

// validBypassTypes is the set of all valid emergency bypass types.
var validBypassTypes = map[string]bool{
	BypassTypeSafetySoundDetected: true,
	BypassTypeHealthCritical:      true,
	BypassTypeCreatorEmergency:    true,
	BypassTypePhysicalHazard:      true,
}

// BypassRequest represents an emergency bypass request (SPEC 14 Section 14.3).
type BypassRequest struct {
	PluginID   string      `json:"plugin_id"`
	BypassType string      `json:"bypass_type"` // safety_sound_detected|health_critical|creator_emergency|physical_hazard
	Payload    interface{} `json:"payload"`
	Timestamp  time.Time   `json:"timestamp"`
}

// BypassResponse is returned after processing a bypass request.
type BypassResponse struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

// Channel handles emergency bypass requests per SPEC 14.
type Channel struct {
	registry *registry.Registry
	bus      *eventbus.Bus
	handlers map[string]func(BypassRequest)
	sub      *nats.Subscription
}

// NewChannel creates a new emergency bypass channel.
func NewChannel(reg *registry.Registry, bus *eventbus.Bus) *Channel {
	return &Channel{
		registry: reg,
		bus:      bus,
		handlers: make(map[string]func(BypassRequest)),
	}
}

// RegisterHandler registers a handler for a specific bypass type.
// Only the four valid bypass types from SPEC 14 Section 14.2 are accepted.
func (c *Channel) RegisterHandler(bypassType string, handler func(BypassRequest)) error {
	if !validBypassTypes[bypassType] {
		return fmt.Errorf("invalid bypass type: %s", bypassType)
	}
	c.handlers[bypassType] = handler
	return nil
}

// HandleBypass processes an emergency bypass request (SPEC 14 Section 14.3).
//
// Steps:
//  1. Validate the bypass type is one of the four allowed types.
//  2. Validate the plugin is authorized for this bypass type via registry.
//  3. Dispatch to the registered handler for the bypass type.
//  4. Publish an emergency event on the eventbus for audit.
//  5. Log the bypass for abuse detection (SPEC 14 Section 14.4).
func (c *Channel) HandleBypass(req BypassRequest) (*BypassResponse, error) {
	// Step 1: Validate bypass type
	if !validBypassTypes[req.BypassType] {
		return &BypassResponse{
			Accepted: false,
			Reason:   fmt.Sprintf("invalid bypass type: %s", req.BypassType),
		}, nil
	}

	// Step 2: Validate plugin authorization via registry
	if err := c.registry.ValidateEmergencyBypass(req.PluginID, req.BypassType); err != nil {
		return &BypassResponse{
			Accepted: false,
			Reason:   err.Error(),
		}, nil
	}

	// Step 3: Dispatch to registered handler (if one exists)
	if handler, ok := c.handlers[req.BypassType]; ok {
		handler(req)
	}

	// Step 4: Publish emergency event on eventbus for audit trail
	if err := c.bus.PublishJSON(EmergencyBypassSubject, req); err != nil {
		log.Printf("bypass: failed to publish emergency event: %v", err)
	}

	// Step 5: Log the bypass for abuse detection (SPEC 14 Section 14.4)
	log.Printf("bypass: emergency bypass accepted — plugin=%s type=%s timestamp=%s",
		req.PluginID, req.BypassType, req.Timestamp.Format(time.RFC3339))

	return &BypassResponse{
		Accepted: true,
		Reason:   "bypass accepted",
	}, nil
}

// Start subscribes to the NATS subject for incoming bypass requests from plugins.
// Plugins publish BypassRequest messages to kognis.emergency.bypass.request.
func (c *Channel) Start() error {
	var err error
	c.sub, err = c.bus.Subscribe(EmergencyBypassRequestSubject, func(msg *nats.Msg) {
		var req BypassRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("bypass: failed to unmarshal request: %v", err)
			return
		}

		resp, _ := c.HandleBypass(req)

		// Send response if the request used reply-to
		if msg.Reply != "" && resp != nil {
			data, err := json.Marshal(resp)
			if err != nil {
				log.Printf("bypass: failed to marshal response: %v", err)
				return
			}
			if err := c.bus.Publish(msg.Reply, data); err != nil {
				log.Printf("bypass: failed to publish response: %v", err)
			}
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", EmergencyBypassRequestSubject, err)
	}
	return nil
}