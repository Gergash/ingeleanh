package api

import (
	"context"
	"fmt"
	"time"

	"github.com/ingeleanh/c2-blockchain/internal/sim"
	"github.com/ingeleanh/c2-blockchain/internal/store"
)

// DemoGatewayID is the simulated residential gateway used for jurado demo data.
const DemoGatewayID = "a1000000-0000-4000-8000-000000000001"

// SeedDemoData populates simulated IoT devices and events for hackathon jurado portal.
func (s *Server) SeedDemoData(ctx context.Context) error {
	now := time.Now()
	lb := now
	if err := s.db.UpsertAgent(ctx, store.Agent{
		ID:        DemoGatewayID,
		ECDSAPub:  "demo-ecdsa-pub-gateway-residencial",
		Hostname:  "gateway-residencial-sim",
		OS:        "linux-amd64",
		Status:    "active",
		AgentRole: "iot_gateway",
		GatewayID: "gw-residencial-01",
		FirstSeen: now.Add(-2 * time.Hour),
		LastBeacon: &lb,
	}); err != nil {
		return err
	}

	devices := []struct {
		id, dtype, zone, lock string
	}{
		{"sensor-access-gate", "access_reader", "porteria-laureles", ""},
		{"sensor-motion-01", "motion_sensor", "lobby", ""},
		{"meter-energy-01", "energy_meter", "unit-101", ""},
		{"lock-main", "smart_lock", "entrance", "locked"},
	}
	for _, d := range devices {
		if err := s.db.UpsertIoTDevice(ctx, d.id, DemoGatewayID, d.dtype, d.zone, d.lock); err != nil {
			return err
		}
	}

	motion := s.sim.MotionEvent("lobby")
	motion["gateway_id"] = DemoGatewayID
	if err := s.db.Audit(ctx, "agent:"+DemoGatewayID, "iot_event", "iot", DemoGatewayID, motion); err != nil {
		return err
	}

	telemetry := s.sim.Telemetry("meter-energy-01", 12.4)
	telemetry["gateway_id"] = DemoGatewayID
	if err := s.db.Audit(ctx, "agent:"+DemoGatewayID, "iot_telemetry", "iot", DemoGatewayID, telemetry); err != nil {
		return err
	}

	motion2 := s.sim.MotionEvent("parking")
	motion2["device_id"] = "sensor-motion-02"
	motion2["gateway_id"] = DemoGatewayID
	if err := s.db.Audit(ctx, "agent:"+DemoGatewayID, "iot_event", "iot", DemoGatewayID, motion2); err != nil {
		return err
	}

	return s.seedLaurelesEvents(ctx, 8)
}

func (s *Server) ensureLaurelesFeed() {
	if s.laureles != nil {
		return
	}
	path := s.cfg.DataCSV
	if path == "" {
		path = sim.DefaultLaurelesCSV
	}
	feed, err := sim.LoadLaurelesCSV(path, 500)
	if err != nil {
		return
	}
	s.laureles = feed
}

func (s *Server) seedLaurelesEvents(ctx context.Context, n int) error {
	s.ensureLaurelesFeed()
	if s.laureles == nil {
		return nil
	}
	for i := 0; i < n; i++ {
		ev := s.laureles.NextEvent()
		ev["gateway_id"] = DemoGatewayID
		if err := s.db.Audit(ctx, "agent:"+DemoGatewayID, "iot_event", "iot", DemoGatewayID, ev); err != nil {
			return err
		}
	}
	return nil
}

// ReplayLaurelesAccess emits one access event from the Laureles CSV (simulated sensor).
func (s *Server) ReplayLaurelesAccess(ctx context.Context) (map[string]interface{}, error) {
	s.ensureLaurelesFeed()
	if s.laureles == nil {
		return nil, fmt.Errorf("laureles csv not loaded")
	}
	ev := s.laureles.NextEvent()
	ev["gateway_id"] = DemoGatewayID
	if err := s.db.Audit(ctx, "agent:"+DemoGatewayID, "iot_event", "iot", DemoGatewayID, ev); err != nil {
		return nil, err
	}
	return ev, nil
}
