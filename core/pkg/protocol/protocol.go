package protocol

// Protocol version constants for the gRPC control plane.
const (
	ProtocolVersion = "0.1.0"

	// Subject prefixes for NATS messaging
	SubjectPrefix       = "kognis"
	SubjectPluginPrefix = "kognis.plugin"
	SubjectPipelinePrefix = "kognis.pipeline"

	// Registration subjects
	SubjectRegister   = "kognis.plugin.register"
	SubjectHealth     = "kognis.plugin.health"
	SubjectShutdown   = "kognis.plugin.shutdown"
	SubjectState      = "kognis.plugin.state"

	// Default configuration
	DefaultNATSPort    = 4222
	DefaultSocketPath  = "/tmp/kognis-control.sock"
	DefaultHBInterval  = 10  // seconds
	DefaultRegTimeout  = 5   // seconds
	DefaultShutdownGrace = 30 // seconds
)

// RegistrationRequest is the message a plugin sends to register with the core.
type RegistrationRequest struct {
	PluginID     string   `json:"plugin_id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Slots        []SlotDecl `json:"slots,omitempty"`
}

// SlotDecl declares a pipeline slot a plugin wants to handle.
type SlotDecl struct {
	Pipeline string `json:"pipeline"`
	Slot     string `json:"slot"`
	Priority int    `json:"priority"`
}

// RegistrationResponse is the core daemon's reply to a registration request.
type RegistrationResponse struct {
	PluginID    string `json:"plugin_id"`
	State       string `json:"state"`
	EventBusURL string `json:"event_bus_url"`
	SocketPath  string `json:"socket_path"`
	Error       string `json:"error,omitempty"`
}

// HealthPulse is the periodic health message from a plugin.
type HealthPulse struct {
	PluginID   string `json:"plugin_id"`
	State      string `json:"state"`
	LatencyMS  int    `json:"latency_ms"`
	MemoryMB   int    `json:"memory_mb"`
	Timestamp  string `json:"timestamp"`
}

// ShutdownNotice tells a plugin to begin graceful shutdown.
type ShutdownNotice struct {
	PluginID   string `json:"plugin_id"`
	GracePeriod int   `json:"grace_period_sec"`
	Reason     string `json:"reason,omitempty"`
}