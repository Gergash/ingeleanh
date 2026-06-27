package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.db.ListIoTDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
		"total":   len(devices),
	})
}

func (s *Server) handleDemoSeed(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DemoMode {
		writeError(w, http.StatusForbidden, "DEMO_DISABLED")
		return
	}
	if err := s.SeedDemoData(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "SEED_FAILED")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"gateway_id": DemoGatewayID,
		"message":    "Datos demo cargados: dispositivos IoT + eventos Laureles CSV",
	})
}

func (s *Server) handleDemoReplayAccess(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DemoMode {
		writeError(w, http.StatusForbidden, "DEMO_DISABLED")
		return
	}
	ev, err := s.ReplayLaurelesAccess(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "CSV_NOT_LOADED")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "event": ev})
}

// handleDemoThreeLayer runs DEMO-011: IoT event → C2 agents → blockchain config (one guided flow).
func (s *Server) handleDemoThreeLayer(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DemoMode {
		writeError(w, http.StatusForbidden, "DEMO_DISABLED")
		return
	}
	ctx := r.Context()
	steps := make([]map[string]interface{}, 0, 3)

	ev, err := s.ReplayLaurelesAccess(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "CSV_NOT_LOADED")
		return
	}
	steps = append(steps, map[string]interface{}{
		"layer":   "iot",
		"step":    1,
		"title":   "Evento residencial (Laureles CSV)",
		"status":  "ok",
		"detail":  ev["payload_summary"],
		"device":  ev["device_id"],
	})

	agents, _ := s.db.ListAgents(ctx)
	active := 0
	var sampleHost string
	for _, a := range agents {
		if a.Status == "active" {
			active++
			if sampleHost == "" {
				sampleHost = a.Hostname
			}
		}
	}
	steps = append(steps, map[string]interface{}{
		"layer":  "c2",
		"step":   2,
		"title":  "Canal C2 (agentes + beacon)",
		"status": "ok",
		"detail": fmt.Sprintf("%d agentes activos; ejemplo host %s", active, sampleHost),
	})

	chainStatus := s.chain.Status()
	if s.cfg.RPCURL != "" && s.cfg.RegistryAddress != "" {
		if cfg, err := s.chain.GetConfig(ctx); err == nil {
			chainStatus["config_version"] = cfg.Version
			chainStatus["beacon_interval_sec"] = cfg.BeaconIntervalSec
			chainStatus["endpoint_hash"] = cfg.EndpointHash
		}
	}
	steps = append(steps, map[string]interface{}{
		"layer":   "blockchain",
		"step":    3,
		"title":   "Registry Polygon Amoy (getConfig)",
		"status":  "ok",
		"detail":  fmt.Sprintf("config v%v beacon %vs", chainStatus["config_version"], chainStatus["beacon_interval_sec"]),
		"hash":    chainStatus["endpoint_hash"],
		"network": chainStatus["network"],
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"demo_id":   "DEMO-011",
		"steps":     steps,
		"chain":     chainStatus,
		"narrative": "IoT simulado → auditoría C2 → verificación on-chain sin URL en claro",
	})
}

func (s *Server) handlePlatforms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"supported": []string{"linux-amd64", "linux-arm64", "windows-amd64"},
		"build":     "scripts/build-agents.sh o scripts/build-agents.ps1 → dist/",
		"note":      "MVP-07: mismo código Go; executor adapta shell por GOOS",
	})
}

func (s *Server) handlePortalInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"demo_mode":        s.cfg.DemoMode,
		"registry_address": s.cfg.RegistryAddress,
		"demo_gateway_id":  DemoGatewayID,
		"version":          "0.1.0",
		"product":          "C2 Blockchain-Blindado — Centro Inteligencia Residencial",
	})
}

func (s *Server) handleDeviceCommand(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	var body struct {
		Action      string `json:"action"`
		DurationSec int    `json:"duration_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	action := strings.TrimSpace(body.Action)
	if action == "" {
		writeError(w, http.StatusBadRequest, "INVALID_COMMAND")
		return
	}
	gatewayID := DemoGatewayID
	if devs, err := s.db.ListIoTDevices(r.Context()); err == nil {
		for _, d := range devs {
			if d["device_id"] == deviceID {
				if g, ok := d["gateway_id"].(string); ok && g != "" {
					gatewayID = g
				}
				break
			}
		}
	}
	res, err := s.sim.LockCommand(deviceID, action, body.DurationSec)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_COMMAND")
		return
	}
	state, _ := res["state"].(string)
	s.db.UpsertIoTDevice(r.Context(), deviceID, gatewayID, "smart_lock", "entrance", state)
	s.redis.SetLockState(r.Context(), deviceID, state)
	meta := res
	meta["device_id"] = deviceID
	meta["gateway_id"] = gatewayID
	meta["action"] = action
	s.db.Audit(r.Context(), "operator:portal", "iot_command_result", "iot", deviceID, meta)
	writeJSON(w, http.StatusOK, res)
}
