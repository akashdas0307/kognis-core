package controlplane

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// ControlPlaneService implements the ControlPlane gRPC service per SPEC 04/05.
// It delegates lifecycle operations to HandshakeManager and queries the
// registry and health aggregator for status reporting.
type ControlPlaneService struct {
	UnimplementedControlPlaneServer

	registry  *registry.Registry
	bus       *eventbus.Bus
	healthAgg *health.Aggregator
	handshake *HandshakeManager
}

// NewControlPlaneService creates a new ControlPlaneService.
// The healthAgg parameter may be nil if the health aggregator is not yet available;
// HealthCheck will report status from the registry only in that case.
func NewControlPlaneService(
	reg *registry.Registry,
	bus *eventbus.Bus,
	healthAgg *health.Aggregator,
	hm *HandshakeManager,
) *ControlPlaneService {
	return &ControlPlaneService{
		registry:  reg,
		bus:       bus,
		healthAgg: healthAgg,
		handshake: hm,
	}
}

// Register handles plugin registration via the handshake protocol (SPEC 04 Section 4.2).
// Steps 1->2: Plugin sends REGISTER_REQUEST, core responds with REGISTER_ACK.
func (s *ControlPlaneService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if s.handshake == nil {
		return nil, status.Error(codes.Internal, "handshake manager not configured")
	}

	handshakeReq := &HandshakeRequest{
		PluginID:             req.PluginId,
		Name:                 req.Name,
		Version:              req.Version,
		Capabilities:         req.Capabilities,
		ManifestHash:         req.ManifestHash,
		EmergencyBypassTypes: req.EmergencyBypassTypes,
		PID:                  int(req.Pid),
		Entrypoint:           req.Entrypoint,
	}

	resp, err := s.handshake.StartHandshake(handshakeReq)
	if err != nil {
		// Return the response with error field populated rather than a gRPC error
		// so the client can read the error details.
		if resp != nil {
			return &RegisterResponse{
				PluginId:         resp.PluginID,
				Error:            resp.Error,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "handshake failed: %v", err)
	}

	return &RegisterResponse{
		PluginId:         resp.PluginID,
		PluginIdRuntime:  resp.PluginIDRuntime,
		State:            resp.State,
		EventBusUrl:      resp.EventBusURL,
		EventBusToken:    resp.EventBusToken,
		ControlPlane:     resp.ControlPlane,
		ConfigBundle:      resp.ConfigBundle,
		PeerCapabilities:  resp.PeerCapabilities,
	}, nil
}

// HealthCheck returns the health status of a specific plugin (SPEC 05/18).
func (s *ControlPlaneService) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	if req.PluginId == "" {
		return nil, status.Error(codes.InvalidArgument, "plugin_id is required")
	}

	entry, ok := s.registry.Get(req.PluginId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "plugin %s not found", req.PluginId)
	}

	resp := &HealthCheckResponse{
		State: string(entry.State),
	}

	// If the health aggregator is available, use it for the health status.
	if s.healthAgg != nil {
		if pulse, found := s.healthAgg.GetPulse(req.PluginId); found {
			resp.Status = pulse.Status
		} else {
			// No health pulse recorded; derive status from registry state.
			resp.Status = stateToHealthStatus(entry.State)
		}
	} else {
		// Without the aggregator, derive health status from registry state.
		resp.Status = stateToHealthStatus(entry.State)
	}

	return resp, nil
}

// ListPlugins returns all registered plugins (SPEC 05).
func (s *ControlPlaneService) ListPlugins(ctx context.Context, req *ListPluginsRequest) (*ListPluginsResponse, error) {
	plugins := s.registry.List()

	result := make([]*PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		result = append(result, &PluginInfo{
			Id:      p.ID,
			Name:    p.Name,
			Version: p.Version,
			State:   string(p.State),
		})
	}

	return &ListPluginsResponse{Plugins: result}, nil
}

// Shutdown initiates graceful shutdown of a plugin (SPEC 04 Section 4.3).
func (s *ControlPlaneService) Shutdown(ctx context.Context, req *ShutdownPluginRequest) (*ShutdownPluginResponse, error) {
	if req.PluginId == "" {
		return nil, status.Error(codes.InvalidArgument, "plugin_id is required")
	}

	// Verify plugin exists
	if _, ok := s.registry.Get(req.PluginId); !ok {
		return nil, status.Errorf(codes.NotFound, "plugin %s not found", req.PluginId)
	}

	gracePeriod := time.Duration(req.GracePeriodSeconds) * time.Second
	if gracePeriod <= 0 {
		gracePeriod = 30 * time.Second // default per SPEC 04
	}

	// Use HandshakeManager if available (full shutdown protocol).
	if s.handshake != nil {
		if err := s.handshake.InitiateShutdown(req.PluginId, gracePeriod); err != nil {
			return &ShutdownPluginResponse{
				Accepted: false,
				Reason:   fmt.Sprintf("shutdown rejected: %v", err),
			}, nil
		}
		return &ShutdownPluginResponse{
			Accepted: true,
			Reason:   fmt.Sprintf("shutdown initiated with grace period %s", gracePeriod),
		}, nil
	}

	// Fallback: direct registry shutdown without the full protocol.
	if err := s.registry.RequestShutdown(req.PluginId); err != nil {
		return &ShutdownPluginResponse{
			Accepted: false,
			Reason:   fmt.Sprintf("shutdown rejected: %v", err),
		}, nil
	}

	return &ShutdownPluginResponse{
		Accepted: true,
		Reason:   fmt.Sprintf("shutdown requested with grace period %s", gracePeriod),
	}, nil
}

// Ready handles the Step 3 signal from a plugin after it has connected to the
// event bus and is ready to process messages (SPEC 04 Section 4.2).
func (s *ControlPlaneService) Ready(ctx context.Context, req *ReadyRequest) (*ReadyAck, error) {
	if s.handshake == nil {
		return nil, status.Error(codes.Internal, "handshake manager not configured")
	}

	readyMsg := &ReadyMessage{
		PluginID: req.PluginId,
	}

	if err := s.handshake.CompleteHandshake(req.PluginId, readyMsg); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to complete handshake: %v", err)
	}

	return &ReadyAck{
		PluginId: req.PluginId,
		Status:   "ACTIVE",
	}, nil
}

// stateToHealthStatus maps a registry PluginState to a health status string
// per SPEC 18 Section 18.1 when no pulse data is available.
func stateToHealthStatus(state registry.PluginState) string {
	switch state {
	case registry.StateHealthyActive:
		return "HEALTHY"
	case registry.StateUnhealthy:
		return "ERROR"
	case registry.StateUnresponsive:
		return "UNRESPONSIVE"
	case registry.StateShuttingDown, registry.StateShutDown:
		return "UNRESPONSIVE"
	case registry.StateStarting, registry.StateRegistered:
		return "HEALTHY" // starting/registered plugins are considered healthy
	case registry.StateCircuitOpen, registry.StateDead:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}