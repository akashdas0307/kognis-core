package envelope

import (
	"encoding/json"
	"fmt"
	"time"
)

// Type indicates the kind of message in the envelope.
type Type string

const (
	TypeCognition   Type = "COGNITION"
	TypePerception  Type = "PERCEPTION"
	TypeAction      Type = "ACTION"
	TypeReflection  Type = "REFLECTION"
	TypeDream       Type = "DREAM"
	TypeMeta        Type = "META"
	TypeControl     Type = "CONTROL"
	TypeHeartbeat   Type = "HEARTBEAT"
	TypeState       Type = "STATE"
	TypeError       Type = "ERROR"
)

// Envelope is the universal message container for all inter-component communication.
// Spec: docs/spec/01-message-envelope.md
type Envelope struct {
	ID          string            `json:"id"`
	Type        Type              `json:"type"`
	Source      string            `json:"source"`
	Destination string            `json:"destination,omitempty"`
	Pipeline    string            `json:"pipeline,omitempty"`
	Slot        string            `json:"slot,omitempty"`
	HopCount    int               `json:"hop_count"`
	MaxHops     int               `json:"max_hops"`
	Priority    string            `json:"priority"`
	Timestamp   time.Time         `json:"timestamp"`
	Payload     json.RawMessage   `json:"payload"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ParseEnvelope deserializes a JSON byte slice into an Envelope.
func ParseEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	return &env, nil
}

// Serialize converts an Envelope to JSON bytes.
func (e *Envelope) Serialize() ([]byte, error) {
	return json.Marshal(e)
}

// Validate checks envelope integrity per spec constraints.
func (e *Envelope) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("envelope ID is required")
	}
	if e.Type == "" {
		return fmt.Errorf("envelope type is required")
	}
	if e.Source == "" {
		return fmt.Errorf("envelope source is required")
	}
	if e.MaxHops > 0 && e.HopCount > e.MaxHops {
		return fmt.Errorf("hop_count %d exceeds max_hops %d", e.HopCount, e.MaxHops)
	}
	return nil
}

// IncrementHop increments the hop count. Returns error if max exceeded.
func (e *Envelope) IncrementHop() error {
	e.HopCount++
	if e.MaxHops > 0 && e.HopCount > e.MaxHops {
		return fmt.Errorf("hop_count %d exceeds max_hops %d", e.HopCount, e.MaxHops)
	}
	return nil
}