package envelope

import (
	"encoding/json"
	"testing"
)

func TestParseEnvelope(t *testing.T) {
	raw := map[string]interface{}{
		"id":       "env-001",
		"type":     "COGNITION",
		"source":   "test-plugin",
		"hop_count": 0,
		"max_hops":  10,
		"priority": "HIGH",
	}
	data, _ := json.Marshal(raw)

	env, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("ParseEnvelope() failed: %v", err)
	}
	if env.ID != "env-001" {
		t.Fatalf("expected ID env-001, got %s", env.ID)
	}
	if env.Type != TypeCognition {
		t.Fatalf("expected type COGNITION, got %s", env.Type)
	}
	if env.Source != "test-plugin" {
		t.Fatalf("expected source test-plugin, got %s", env.Source)
	}
}

func TestEnvelopeValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     *Envelope
		wantErr bool
	}{
		{
			name: "valid envelope",
			env: &Envelope{
				ID: "env-002", Type: TypeAction, Source: "src",
				HopCount: 0, MaxHops: 10, Priority: "MEDIUM",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			env: &Envelope{
				Type: TypeAction, Source: "src",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			env: &Envelope{
				ID: "env-003", Source: "src",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			env: &Envelope{
				ID: "env-004", Type: TypeAction,
			},
			wantErr: true,
		},
		{
			name: "hop count exceeded",
			env: &Envelope{
				ID: "env-005", Type: TypeAction, Source: "src",
				HopCount: 11, MaxHops: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIncrementHop(t *testing.T) {
	env := &Envelope{
		ID: "env-006", Type: TypeCognition, Source: "src",
		HopCount: 0, MaxHops: 3, Priority: "MEDIUM",
	}

	for i := 1; i <= 3; i++ {
		if err := env.IncrementHop(); err != nil {
			t.Fatalf("IncrementHop() %d failed: %v", i, err)
		}
		if env.HopCount != i {
			t.Fatalf("expected hop_count %d, got %d", i, env.HopCount)
		}
	}

	// 4th hop should fail
	if err := env.IncrementHop(); err == nil {
		t.Fatal("expected error for exceeding max_hops, got nil")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	env := &Envelope{
		ID: "env-007", Type: TypeReflection, Source: "src",
		HopCount: 0, MaxHops: 5, Priority: "LOW",
	}

	data, err := env.Serialize()
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	parsed, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("ParseEnvelope() failed: %v", err)
	}
	if parsed.ID != env.ID {
		t.Fatalf("round-trip: expected ID %s, got %s", env.ID, parsed.ID)
	}
	if parsed.Type != env.Type {
		t.Fatalf("round-trip: expected type %s, got %s", env.Type, parsed.Type)
	}
}