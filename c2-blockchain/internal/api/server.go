package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ingeleanh/c2-blockchain/internal/chain"
	"github.com/ingeleanh/c2-blockchain/internal/config"
	"github.com/ingeleanh/c2-blockchain/internal/crypto"
	"github.com/ingeleanh/c2-blockchain/internal/executor"
	"github.com/ingeleanh/c2-blockchain/internal/handshake"
	"github.com/ingeleanh/c2-blockchain/internal/sim"
	"github.com/ingeleanh/c2-blockchain/internal/store"
)

type Server struct {
	cfg         config.Config
	db          *store.SQLite
	redis       *store.RedisStore
	handshake   *handshake.Service
	chain       *chain.Reader
	sim         *sim.Simulator
	masterKey   []byte
	challenges  map[string]*handshake.Challenge
	sessionKeys map[string][]byte
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

func NewServer(cfg config.Config, db *store.SQLite, redis *store.RedisStore, chainReader *chain.Reader) *Server {
	nonceStore := store.NewRedisNonceStore(redis)
	mk, _ := store.MasterKeyFromHex(cfg.MasterKeyHex)
	return &Server{
		cfg:         cfg,
		db:          db,
		redis:       redis,
		handshake:   handshake.NewService(nonceStore),
		chain:       chainReader,
		sim:         sim.NewSimulator(),
		masterKey:   mk,
		challenges:  make(map[string]*handshake.Challenge),
		sessionKeys: make(map[string][]byte),
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", s.handleHealth)
	r.Get("/api/v1/health", s.handleHealth)
	r.Post("/api/v1/operator/login", s.handleOperatorLogin)
	r.Post("/api/v1/agents/handshake", s.handleHandshake)
	r.Post("/api/v1/agents/{agentID}/beacon", s.handleBeacon)
	r.Post("/api/v1/agents/{agentID}/task_result", s.handleTaskResult)
	r.Get("/api/v1/agents", s.auth(s.handleListAgents))
	r.Post("/api/v1/tasks", s.auth(s.handleCreateTask))
	r.Get("/api/v1/tasks/{taskID}", s.auth(s.handleGetTask))
	r.Get("/api/v1/events", s.auth(s.handleEvents))
	r.Get("/api/v1/devices/{deviceID}/state", s.auth(s.handleDeviceState))
	r.Get("/api/v1/chain/status", s.auth(s.handleChainStatus))
	r.Get("/api/v1/ws/agent", s.handleWSAgent)
	r.Handle("/dashboard/*", http.StripPrefix("/dashboard/", http.FileServer(http.Dir("frontend/dashboard"))))
	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/", http.StatusFound)
	})
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	chainOK := s.redis.Ping(r.Context()) == nil
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "ok",
		"version":          "0.1.0",
		"chain_connected":  chainOK,
		"registry_address": s.cfg.RegistryAddress,
	})
}

func (s *Server) handleOperatorLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	if body.Username != s.cfg.OperatorUser || body.Password != s.cfg.OperatorPass {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED")
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": body.Username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TOKEN_ERROR")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": signed})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.cfg.JWTSecret), nil
		})
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHandshake(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	count, _ := s.redis.IncrRate(r.Context(), "rate:ip:"+ip, time.Minute)
	if count > 10 {
		writeError(w, http.StatusTooManyRequests, "RATE_LIMITED")
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	step, _ := body["step"].(string)
	switch step {
	case "challenge_request":
		ch, err := s.handshake.CreateChallenge()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL")
			return
		}
		s.mu.Lock()
		s.challenges[ch.Nonce] = ch
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"step":            "challenge",
			"nonce":           ch.Nonce,
			"server_ecdh_pub": ch.ServerECDHPub,
			"expires_at":      ch.ExpiresAt,
		})
	case "challenge_response":
		nonce, _ := body["nonce"].(string)
		s.mu.Lock()
		ch := s.challenges[nonce]
		s.mu.Unlock()
		if ch == nil {
			writeError(w, http.StatusBadRequest, "NONCE_EXPIRED")
			return
		}
		ts, _ := body["timestamp"].(float64)
		resp := &handshake.ChallengeResponse{
			Nonce:         nonce,
			AgentECDSAPub: fmt.Sprint(body["agent_ecdsa_pub"]),
			AgentECDHPub:  fmt.Sprint(body["agent_ecdh_pub"]),
			Signature:     fmt.Sprint(body["signature"]),
			Hostname:      fmt.Sprint(body["hostname"]),
			OS:            fmt.Sprint(body["os"]),
			Timestamp:     int64(ts),
		}
		agentRole := "generic"
		if body["agent_role"] == "iot_gateway" {
			agentRole = "iot_gateway"
		}
		result, err := s.handshake.CompleteChallenge(ch, resp)
		if err != nil {
			code := "SIGNATURE_INVALID"
			if err == handshake.ErrNonceExpired {
				code = "NONCE_EXPIRED"
			}
			if err == handshake.ErrNonceReused {
				code = "NONCE_REUSED"
			}
			writeError(w, http.StatusBadRequest, code)
			return
		}
		agentID := uuid.New().String()
		sessionID := "sess_" + uuid.New().String()[:8]
		now := time.Now()
		agent := store.Agent{
			ID:        agentID,
			ECDSAPub:  resp.AgentECDSAPub,
			Hostname:  resp.Hostname,
			OS:        resp.OS,
			Status:    "active",
			AgentRole: agentRole,
			FirstSeen: now,
		}
		if gw, ok := body["gateway_id"].(string); ok {
			agent.GatewayID = gw
		}
		if err := s.db.UpsertAgent(r.Context(), agent); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL")
			return
		}
		expires := now.Add(24 * time.Hour)
		sess := store.Session{ID: sessionID, AgentID: agentID, Established: now, ExpiresAt: expires, Status: "active"}
		if err := s.db.CreateSession(r.Context(), sess, result.SessionKey); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL")
			return
		}
		s.mu.Lock()
		s.sessionKeys[sessionID] = result.SessionKey
		s.mu.Unlock()
		s.db.Audit(r.Context(), "agent:"+agentID, "handshake", "agent", agentID, nil)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"step":               "complete",
			"agent_id":           agentID,
			"session_key_id":     sessionID,
			"session_expires_at": expires.Unix(),
			"status":             "registered",
		})
	default:
		writeError(w, http.StatusBadRequest, "INVALID_STEP")
	}
}

func (s *Server) sessionKey(sessionID string) ([]byte, bool) {
	s.mu.RLock()
	k, ok := s.sessionKeys[sessionID]
	s.mu.RUnlock()
	return k, ok
}

func (s *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	sessionID := r.Header.Get("X-Session-Key-Id")
	key, ok := s.sessionKey(sessionID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "SESSION_INVALID")
		return
	}
	var req struct {
		Encrypted crypto.Envelope `json:"encrypted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY")
		return
	}
	plain, err := req.Encrypted.Decrypt(key)
	if err != nil {
		writeError(w, http.StatusBadRequest, "DECRYPT_FAILED")
		return
	}
	var beacon map[string]interface{}
	json.Unmarshal(plain, &beacon)
	s.db.UpdateLastBeacon(r.Context(), agentID, time.Now())

	msgType, _ := beacon["type"].(string)
	if msgType == "iot_event" || msgType == "iot_telemetry" {
		action := "iot_event"
		if msgType == "iot_telemetry" {
			action = "iot_telemetry"
		}
		meta := beacon
		meta["gateway_id"] = agentID
		s.db.Audit(r.Context(), "agent:"+agentID, action, "iot", agentID, meta)
	}

	task, _ := s.redis.PopTask(r.Context(), agentID)
	var respPlain []byte
	if task != nil {
		respPlain, _ = json.Marshal(map[string]interface{}{
			"type":         "task",
			"task_id":      task["task_id"],
			"command_type": task["command_type"],
			"payload":      task["payload"],
		})
	} else {
		respPlain, _ = json.Marshal(map[string]interface{}{
			"type":      "ack",
			"timestamp": time.Now().Unix(),
		})
	}
	var enc crypto.Envelope
	if err := enc.Encrypt(key, respPlain); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"encrypted": enc})
}

func (s *Server) handleTaskResult(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	sessionID := r.Header.Get("X-Session-Key-Id")
	key, ok := s.sessionKey(sessionID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "SESSION_INVALID")
		return
	}
	var req struct {
		Encrypted crypto.Envelope `json:"encrypted"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	plain, err := req.Encrypted.Decrypt(key)
	if err != nil {
		writeError(w, http.StatusBadRequest, "DECRYPT_FAILED")
		return
	}
	var result map[string]interface{}
	json.Unmarshal(plain, &result)
	taskID, _ := result["task_id"].(string)
	exitCode := 0
	if ec, ok := result["exit_code"].(float64); ok {
		exitCode = int(ec)
	}
	stdout, _ := result["stdout"].(string)
	stderr, _ := result["stderr"].(string)
	if result["type"] == "iot_command_result" || result["device_id"] != nil {
		s.db.Audit(r.Context(), "agent:"+agentID, "iot_command_result", "iot", taskID, result)
		if devID, ok := result["device_id"].(string); ok {
			if state, ok := result["state"].(string); ok {
				s.db.UpsertIoTDevice(r.Context(), devID, agentID, "smart_lock", "", state)
				s.redis.SetLockState(r.Context(), devID, state)
			}
		}
		stdout = s.sim.ToJSON(result)
		exitCode = 0
	}
	s.db.CompleteTask(r.Context(), taskID, exitCode, stdout, stderr)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "task_id": taskID})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.db.ListAgents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL")
		return
	}
	out := make([]map[string]interface{}, 0, len(agents))
	for _, a := range agents {
		m := map[string]interface{}{
			"agent_id":   a.ID,
			"hostname":   a.Hostname,
			"os":         a.OS,
			"status":     a.Status,
			"agent_role": a.AgentRole,
			"first_seen": a.FirstSeen.Format(time.RFC3339),
		}
		if a.LastBeacon != nil {
			m["last_beacon"] = a.LastBeacon.Format(time.RFC3339)
		}
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": out, "total": len(out)})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentID     string                 `json:"agent_id"`
		CommandType string                 `json:"command_type"`
		Payload     map[string]interface{} `json:"payload"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	taskID := uuid.New().String()
	payloadJSON, _ := json.Marshal(body.Payload)
	task := store.Task{
		ID:          taskID,
		AgentID:     body.AgentID,
		CommandType: body.CommandType,
		PayloadJSON: string(payloadJSON),
		Status:      "pending",
		CreatedBy:   "operator",
		CreatedAt:   time.Now(),
	}
	s.db.CreateTask(r.Context(), task)
	s.redis.PushTask(r.Context(), body.AgentID, map[string]interface{}{
		"task_id":      taskID,
		"command_type": body.CommandType,
		"payload":      body.Payload,
	})
	writeJSON(w, http.StatusCreated, map[string]string{"task_id": taskID, "status": "pending"})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.db.GetTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id":    t.ID,
		"agent_id":   t.AgentID,
		"status":     t.Status,
		"exit_code":  t.ExitCode,
		"stdout":     t.Stdout,
		"stderr":     t.Stderr,
		"created_at": t.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	gatewayID := r.URL.Query().Get("gateway_id")
	events, total, err := s.db.ListIoTEvents(r.Context(), gatewayID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events, "total": total, "limit": limit, "offset": offset,
	})
}

func (s *Server) handleDeviceState(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	deviceType, lockState, err := s.db.GetIoTDeviceState(r.Context(), deviceID)
	if err != nil {
		lockState = s.sim.GetLockState(deviceID)
		deviceType = "smart_lock"
	}
	if ls, err := s.redis.GetLockState(r.Context(), deviceID); err == nil && ls != "" {
		lockState = ls
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":   deviceID,
		"device_type": deviceType,
		"state":       lockState,
		"updated_at":  time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleChainStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.chain.Status())
}

func (s *Server) handleWSAgent(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var frame map[string]interface{}
		json.Unmarshal(msg, &frame)
		agentID, _ := frame["agent_id"].(string)
		sessionID, _ := frame["session_key_id"].(string)
		key, ok := s.sessionKey(sessionID)
		if !ok {
			continue
		}
		if enc, ok := frame["encrypted"].(map[string]interface{}); ok {
			var envelope crypto.Envelope
			b, _ := json.Marshal(enc)
			json.Unmarshal(b, &envelope)
			plain, err := envelope.Decrypt(key)
			if err != nil {
				continue
			}
			var beacon map[string]interface{}
			json.Unmarshal(plain, &beacon)
			s.db.UpdateLastBeacon(context.Background(), agentID, time.Now())
			task, _ := s.redis.PopTask(context.Background(), agentID)
			var respPlain []byte
			if task != nil {
				respPlain, _ = json.Marshal(map[string]interface{}{
					"type":         "task",
					"task_id":      task["task_id"],
					"command_type": task["command_type"],
					"payload":      task["payload"],
				})
			} else {
				respPlain, _ = json.Marshal(map[string]interface{}{"type": "ack", "timestamp": time.Now().Unix()})
			}
			var out crypto.Envelope
			out.Encrypt(key, respPlain)
			conn.WriteJSON(map[string]interface{}{"encrypted": out})
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error_code": code})
}

// ExecuteTaskOnAgent runs shell or iot_command locally (agent side).
func ExecuteTaskOnAgent(ctx context.Context, sim *sim.Simulator, commandType string, payload map[string]interface{}) executor.Result {
	switch commandType {
	case "shell":
		argv, _ := payload["argv"].([]interface{})
		strs := make([]string, len(argv))
		for i, v := range argv {
			strs[i] = fmt.Sprint(v)
		}
		res, _ := executor.Shell(ctx, strs)
		return res
	case "iot_command":
		device, _ := payload["target_device"].(string)
		action, _ := payload["action"].(string)
		dur, _ := payload["duration_sec"].(float64)
		out, err := sim.LockCommand(device, action, int(dur))
		if err != nil {
			return executor.Result{ExitCode: 1, Stderr: err.Error()}
		}
		b, _ := json.Marshal(out)
		return executor.Result{ExitCode: 0, Stdout: string(b)}
	default:
		return executor.Result{ExitCode: 1, Stderr: "unknown command"}
	}
}
