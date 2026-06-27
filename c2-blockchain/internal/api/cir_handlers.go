package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const laurelesRecordsValidated = 119186

func (s *Server) handleCIRInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"default_user": s.cfg.OperatorUser,
		"demo_mode":    s.cfg.DemoMode,
		"portal_path":  "/portal/",
	})
}

func (s *Server) handleCIRSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildCIRSummary(r))
}

func (s *Server) handleCIRAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": s.buildCIRAlerts(r),
	})
}

func (s *Server) handleCIRPulse(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pulse": s.buildCIRPulse(r),
	})
}

func (s *Server) handleCIRSimulateAttack(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Attack-Token")
	if token != "compromised" {
		writeError(w, http.StatusUnauthorized, "ATTACK_TOKEN_INVALID")
		return
	}
	full := r.URL.Query().Get("full") == "true"
	alertText := "Intento de inyección API bloqueado — ECDSA rechazó token · registrado on-chain"
	s.db.Audit(r.Context(), "cir:simulator", "security_block", "c2", "simulate-attack", map[string]interface{}{
		"blocked": true, "full": full, "latency_ms": 340,
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"blocked":     true,
		"latency_ms":  340,
		"kpi_alerts":  8,
		"alert_text":  alertText,
		"defense":     "ECDSA + JWT invalid · audit_log + chain trace",
		"simulation":  full,
	})
}

func (s *Server) buildCIRSummary(r *http.Request) map[string]interface{} {
	agents, _ := s.db.ListAgents(r.Context())
	activeNodes := 0
	for _, a := range agents {
		if a.Status == "active" {
			activeNodes++
		}
	}
	if activeNodes == 0 {
		activeNodes = 3
	}
	totalNodes := activeNodes
	if totalNodes < 3 {
		totalNodes = 3
	}

	return map[string]interface{}{
		"records_valid": laurelesRecordsValidated,
		"summary": map[string]interface{}{
			"total_records":   laurelesRecordsValidated,
			"active_nodes":    activeNodes,
			"total_nodes":     totalNodes,
			"security_score":  94.2,
			"user_types":      map[string]int{"residentes_pct": 73, "visitantes_pct": 18, "domicilios_pct": 9},
			"hourly_baseline": cirHourlySeries(false),
			"hourly_today":    cirHourlySeries(true),
			"sensor_counts": map[string]int{
				"Movimiento": 24, "Temperatura": 12, "Consumo": 8, "Cámaras": 16, "Acceso": 6,
			},
			"vuln_patterns": map[string]int{
				"Phishing": 72, "API exploit": 58, "Inyección": 44, "Fuerza bruta": 35, "MitM": 20, "DoS": 48,
			},
			"sec_events_30d": []int{2, 1, 3, 0, 2, 4, 1, 2, 0, 3, 2, 1, 0, 2, 1, 3, 2, 0, 1, 2, 4, 1, 0, 2, 3, 1, 2, 0, 1, 2},
			"weekly_report": []map[string]int{
				{"residentes": 21840, "visitantes": 5040, "domicilios": 2520},
				{"residentes": 28920, "visitantes": 6720, "domicilios": 3360},
				{"residentes": 24480, "visitantes": 5760, "domicilios": 2880},
				{"residentes": 31620, "visitantes": 7560, "domicilios": 3780},
			},
		},
	}
}

func cirHourlySeries(today bool) []map[string]int {
	out := make([]map[string]int, 24)
	for h := 0; h < 24; h++ {
		base := 40 + (h%6)*8
		if h >= 7 && h <= 9 {
			base += 35
		}
		if h >= 17 && h <= 20 {
			base += 50
		}
		if today && h == time.Now().Hour() {
			base += 12
		}
		out[h] = map[string]int{"hour": h, "count": base}
	}
	return out
}

func (s *Server) buildCIRAlerts(r *http.Request) []map[string]interface{} {
	now := time.Now()
	alerts := []map[string]interface{}{
		{
			"severity": "crit",
			"text":     "Acceso atípico torre 5 — volumen +40% vs línea base miércoles",
			"time":     now.Add(-3 * time.Minute).Format(time.RFC3339),
			"source":   "Motor predictivo CIR",
		},
		{
			"severity": "warn",
			"text":     "Vehículo recurrente T06·39 detectado en ventana 72h",
			"time":     now.Add(-11 * time.Minute).Format(time.RFC3339),
			"source":   "Correlación nodos",
		},
		{
			"severity": "ok",
			"text":     "Config on-chain C2Registry v1 verificada — endpointHash OK",
			"time":     now.Add(-25 * time.Minute).Format(time.RFC3339),
			"source":   "Polygon Amoy",
		},
	}
	events, _, _ := s.db.ListIoTEvents(r.Context(), "", 3, 0)
	for _, ev := range events {
		summary, _ := ev["payload_summary"].(string)
		if summary == "" {
			continue
		}
		created, _ := ev["created_at"].(string)
		alerts = append(alerts, map[string]interface{}{
			"severity": "warn",
			"text":     summary,
			"time":     created,
			"source":   "Laureles · sensor-access-gate",
		})
	}
	return alerts
}

func (s *Server) buildCIRPulse(r *http.Request) []map[string]interface{} {
	events, _, _ := s.db.ListIoTEvents(r.Context(), "", 12, 0)
	pulse := make([]map[string]interface{}, 0, len(events))
	for _, ev := range events {
		row := mapCIRPulseRow(ev)
		if row != nil {
			pulse = append(pulse, row)
		}
	}
	if len(pulse) > 0 {
		return pulse
	}
	return cirDefaultPulse()
}

func mapCIRPulseRow(ev map[string]interface{}) map[string]interface{} {
	torre, _ := ev["torre"].(string)
	apto, _ := ev["apto"].(string)
	tipo, _ := ev["tipo_registro"].(string)
	if tipo == "" {
		tipo, _ = ev["device_type"].(string)
	}
	ident := truncateStr(evString(ev, "payload_summary"), 48)
	if ident == "" {
		ident = anonIdentity(ev)
	}
	created, _ := ev["created_at"].(string)
	if created == "" {
		created = time.Now().Format("2006-01-02 15:04:05")
	} else if t, err := time.Parse(time.RFC3339, created); err == nil {
		created = t.Format("2006-01-02 15:04:05")
	}
	torreApto := strings.TrimSpace(torre + " · Apt " + apto)
	if torreApto == "· Apt" {
		torreApto = "Laureles · Portería"
	}
	estado := "Autorizado"
	if strings.Contains(strings.ToLower(tipo), "visit") {
		estado = "Verificado"
	}
	return map[string]interface{}{
		"fecha_hora":  created,
		"torre_apto":  torreApto,
		"tipo":        normalizePulseTipo(tipo),
		"identidad":   ident,
		"estado":      estado,
		"alert":       false,
	}
}

func anonIdentity(ev map[string]interface{}) string {
	placa, _ := ev["placa"].(string)
	if placa != "" {
		return "Placa " + placa
	}
	return "T06·" + strconv.Itoa(10+len(evString(ev, "device_id"))%89)
}

func normalizePulseTipo(tipo string) string {
	t := strings.ToLower(tipo)
	switch {
	case strings.Contains(t, "resident"):
		return "Residente"
	case strings.Contains(t, "visit"):
		return "Visitante"
	case strings.Contains(t, "domicil"):
		return "Domicilio"
	case strings.Contains(t, "access") || strings.Contains(t, "acceso"):
		return "Acceso"
	default:
		if tipo != "" {
			return tipo
		}
		return "Residente"
	}
}

func evString(ev map[string]interface{}, key string) string {
	v, _ := ev[key].(string)
	return v
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func cirDefaultPulse() []map[string]interface{} {
	now := time.Now()
	return []map[string]interface{}{
		{
			"fecha_hora": now.Add(-2 * time.Minute).Format("2006-01-02 15:04:05"),
			"torre_apto": "Torre 3 · Apt 402", "tipo": "Residente",
			"identidad": "T06·39", "estado": "Autorizado", "alert": false,
		},
		{
			"fecha_hora": now.Add(-8 * time.Minute).Format("2006-01-02 15:04:05"),
			"torre_apto": "Torre 1 · Apt 101", "tipo": "Visitante",
			"identidad": "Placa ABC123", "estado": "Verificado", "alert": false,
		},
	}
}
