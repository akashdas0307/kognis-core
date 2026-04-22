// Package capability implements the full capability registry per SPEC 05 Section 5.3.
//
// It provides querying, schema tracking, LLM exposure filtering, and change
// notification on top of the data already held by the registry package.
// The Store does NOT duplicate registry data — it references the registry
// and enriches capability entries with schema/latency/exposure metadata.
package capability

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// CapabilityEntry extends the registry capability concept with schema,
// latency class, and LLM exposure metadata (SPEC 05 Section 5.3).
type CapabilityEntry struct {
	ID          string
	ProviderIDs []string
	Status      string // "available" or "unavailable"
	Schema      map[string]interface{} // JSON schema for params and response
	LatencyClass string               // e.g. "realtime", "batch", "interactive"
	LLMExposedTo []string             // plugin IDs that may invoke this via LLM routing
}

// changeEvent is the payload published on kognis.capability.changed.
type changeEvent struct {
	CapID  string `json:"cap_id"`
	Status string `json:"status"`
}

// Store is the full capability registry. It holds enriched CapabilityEntry
// values and delegates core plugin/capability data to *registry.Registry.
type Store struct {
	mu   sync.RWMutex
	caps map[string]*CapabilityEntry // capID -> enriched entry

	reg *registry.Registry
	bus *eventbus.Bus
}

// NewStore creates a capability store backed by the given registry and event bus.
func NewStore(reg *registry.Registry, bus *eventbus.Bus) *Store {
	return &Store{
		caps: make(map[string]*CapabilityEntry),
		reg:  reg,
		bus:  bus,
	}
}

// QueryAvailable returns true if the capability identified by capID exists
// and has status "available" (SPEC 05 Section 5.3).
func (s *Store) QueryAvailable(capID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.caps[capID]
	if !ok {
		return false
	}
	return entry.Status == "available"
}

// FindProviders returns the plugin IDs that provide the given capability.
// Returns nil if the capability is unknown (SPEC 05 Section 5.3).
func (s *Store) FindProviders(capID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.caps[capID]
	if !ok {
		return nil
	}
	// Return a copy to avoid caller mutating internal state.
	out := make([]string, len(entry.ProviderIDs))
	copy(out, entry.ProviderIDs)
	return out
}

// GetSchema returns the JSON schema for a capability (params + response).
// Returns ErrCapabilityNotFound if the capability does not exist in the store
// (SPEC 05 Section 5.3).
func (s *Store) GetSchema(capID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.caps[capID]
	if !ok {
		return nil, fmt.Errorf("%w: capability %s not found in capability store", registry.ErrCapabilityNotFound, capID)
	}
	// Return a shallow copy of the schema map.
	if entry.Schema == nil {
		return nil, nil
	}
	out := make(map[string]interface{}, len(entry.Schema))
	for k, v := range entry.Schema {
		out[k] = v
	}
	return out, nil
}

// ListForLLM returns capabilities that are exposed to the given requesting
// plugin via LLM routing. A capability is included if the requestingPluginID
// appears in its LLMExposedTo list, or if LLMExposedTo is empty (meaning
// exposed to all) (SPEC 05 Section 5.3).
func (s *Store) ListForLLM(requestingPluginID string) []CapabilityEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []CapabilityEntry
	for _, entry := range s.caps {
		if isExposedTo(entry.LLMExposedTo, requestingPluginID) {
			// Return a value copy.
			result = append(result, *entry)
		}
	}
	return result
}

// SubscribeChanges subscribes to capability change events published on
// kognis.capability.changed. The callback receives the capability ID and
// its new status string ("available" or "unavailable") (SPEC 05 Section 5.3).
func (s *Store) SubscribeChanges(callback func(capID string, status string)) (*nats.Subscription, error) {
	subject := eventbus.CapabilitySubject()

	handler := func(msg *nats.Msg) {
		var evt changeEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			return
		}
		callback(evt.CapID, evt.Status)
	}

	if s.bus == nil {
		return nil, fmt.Errorf("event bus is nil; cannot subscribe to capability changes")
	}
	return s.bus.Subscribe(subject, handler)
}

// SyncFromRegistry rebuilds the capability store from the current state of
// the plugin registry. It is called after plugin registration or removal.
// When a capability is added or its status changes, a change event is
// published on kognis.capability.changed (SPEC 05 Section 5.3).
func (s *Store) SyncFromRegistry() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect current capability data from registry plugins.
	plugins := s.reg.List()
	regCaps := make(map[string]*CapabilityEntry) // rebuilt map

	for _, p := range plugins {
		for _, capID := range p.Capabilities {
			if existing, ok := regCaps[capID]; ok {
				existing.ProviderIDs = append(existing.ProviderIDs, p.ID)
			} else {
				entry := &CapabilityEntry{
					ID:           capID,
					ProviderIDs:  []string{p.ID},
					Status:       capabilityStatusFromPlugin(p),
					Schema:       nil, // populated separately via SetSchema
					LatencyClass: p.LatencyClass,
					LLMExposedTo: copyStrings(p.LLMExposedTo),
				}
				regCaps[capID] = entry
			}
		}
	}

	// Preserve schemas from previous store entries (schemas are set
	// independently via SetSchema, not derived from the registry).
	for capID, oldEntry := range s.caps {
		if newEntry, ok := regCaps[capID]; ok && oldEntry.Schema != nil {
			newEntry.Schema = oldEntry.Schema
		}
	}

	// Detect changes and publish events before replacing the map.
	if s.bus != nil {
		s.publishChanges(s.caps, regCaps)
	}

	s.caps = regCaps
}

// SetSchema sets the JSON schema for a capability's params and response.
// This is called by the plugin SDK or configuration loader, not by SyncFromRegistry.
func (s *Store) SetSchema(capID string, schema map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.caps[capID]
	if !ok {
		return fmt.Errorf("%w: capability %s not found in capability store", registry.ErrCapabilityNotFound, capID)
	}
	entry.Schema = schema
	return nil
}

// publishChanges compares old and new capability maps and publishes a
// kognis.capability.changed event for every added or status-changed capability.
// Must be called with s.mu held for writing.
func (s *Store) publishChanges(oldCaps, newCaps map[string]*CapabilityEntry) {
	subject := eventbus.CapabilitySubject()

	// New capabilities or status changes.
	for capID, newEntry := range newCaps {
		oldEntry, existed := oldCaps[capID]
		if !existed || oldEntry.Status != newEntry.Status {
			evt := changeEvent{CapID: capID, Status: newEntry.Status}
			_ = s.bus.PublishJSON(subject, evt)
		}
	}

	// Removed capabilities — publish "unavailable" for anything that
	// existed before but is gone now.
	for capID := range oldCaps {
		if _, stillExists := newCaps[capID]; !stillExists {
			evt := changeEvent{CapID: capID, Status: "unavailable"}
			_ = s.bus.PublishJSON(subject, evt)
		}
	}
}

// isExposedTo determines whether a capability is exposed to the requesting
// plugin. An empty LLMExposedTo list means exposed to everyone.
func isExposedTo(exposedTo []string, requestingPluginID string) bool {
	if len(exposedTo) == 0 {
		return true
	}
	for _, id := range exposedTo {
		if id == requestingPluginID {
			return true
		}
	}
	return false
}

// capabilityStatusFromPlugin maps a plugin entry to capability status,
// delegating to the registry's own capability status logic.
func capabilityStatusFromPlugin(p *registry.PluginEntry) string {
	switch p.State {
	case registry.StateHealthyActive:
		return "available"
	default:
		return "unavailable"
	}
}

// copyStrings returns a copy of the string slice, or nil if empty.
func copyStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}