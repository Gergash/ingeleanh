package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ingeleanh/c2-blockchain/internal/chain"
	"github.com/ingeleanh/c2-blockchain/internal/config"
	"github.com/ingeleanh/c2-blockchain/internal/crypto"
	"github.com/ingeleanh/c2-blockchain/internal/sim"
	"github.com/ingeleanh/c2-blockchain/internal/store"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T) (*Server, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	tmp := filepath.Join(os.TempDir(), "c2-test-"+time.Now().Format("150405")+".db")
	db, err := store.OpenSQLite(tmp)
	require.NoError(t, err)
	migration := filepath.Join("..", "..", "migrations", "001_initial.sql")
	if _, err := os.Stat(migration); err != nil {
		migration = filepath.Join("migrations", "001_initial.sql")
	}
	require.NoError(t, db.Migrate(migration))
	redis := store.NewRedis(mr.Addr())
	cache := chain.NewCache()
	cache.Update("0xabc", 30, 1, 0)
	reader, _ := chain.NewReader("", "", cache)
	cfg := config.Config{
		JWTSecret:    "test-secret",
		OperatorUser: "operator",
		OperatorPass: "lab",
	}
	srv := NewServer(cfg, db, redis, reader)
	return srv, func() {
		db.Close()
		os.Remove(tmp)
	}
}

func loginToken(t *testing.T, srv *Server) string {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/login", bytes.NewReader([]byte(`{"username":"operator","password":"lab"}`)))
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["token"]
}

func doHandshake(t *testing.T, srv *Server) (agentID, sessionID string, sessionKey []byte) {
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/handshake", bytes.NewReader([]byte(`{"step":"challenge_request"}`)))
	w1 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w1, req1)
	var ch map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &ch)
	nonce := ch["nonce"].(string)
	serverECDHPub := ch["server_ecdh_pub"].(string)

	priv, _ := crypto.GenerateECDSAKeypair()
	agentECDH, _ := crypto.GenerateECDHKeypair()
	sig, _ := crypto.SignNonce(priv, nonce)
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&agentECDH.PublicKey)
	body := map[string]interface{}{
		"step":            "challenge_response",
		"nonce":           nonce,
		"agent_ecdsa_pub": crypto.ECDSAPublicKeyHex(priv),
		"agent_ecdh_pub":  base64.StdEncoding.EncodeToString(pubBytes),
		"signature":       sig,
		"hostname":        "test-vm",
		"os":              "linux-amd64",
		"timestamp":       time.Now().Unix(),
	}
	b, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/handshake", bytes.NewReader(b))
	w2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var complete map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &complete)
	agentID = complete["agent_id"].(string)
	sessionID = complete["session_key_id"].(string)

	serverPubBytes, _ := base64.StdEncoding.DecodeString(serverECDHPub)
	serverPub, _ := crypto.UnmarshalECDHPublicKeySPKI(agentECDH.Curve, serverPubBytes)
	shared, _ := crypto.ECDHSharedSecret(agentECDH, serverPub)
	sessionKey, _ = crypto.DeriveSessionKey(shared, "c2-session-v1")
	return agentID, sessionID, sessionKey
}

func TestAPI001_HandshakeComplete(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	agentID, sessionID, key := doHandshake(t, srv)
	require.NotEmpty(t, agentID)
	require.NotEmpty(t, sessionID)
	require.Len(t, key, 32)
	a, err := srv.db.GetAgent(context.Background(), agentID)
	require.NoError(t, err)
	require.Equal(t, "active", a.Status)
}

func TestAPI003_OperatorWithoutJWT(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI004_CreateTask(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	agentID, _, _ := doHandshake(t, srv)
	token := loginToken(t, srv)
	body, _ := json.Marshal(map[string]interface{}{
		"agent_id":     agentID,
		"command_type": "shell",
		"payload":      map[string]interface{}{"argv": []string{"whoami"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestAPI006_RateLimitHandshake(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	var lastCode int
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/handshake", bytes.NewReader([]byte(`{"step":"challenge_request"}`)))
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		lastCode = w.Code
	}
	require.Equal(t, http.StatusTooManyRequests, lastCode)
}

func TestAPI007_ListEvents(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	agentID, sessionID, key := doHandshake(t, srv)
	plain, _ := json.Marshal(map[string]interface{}{
		"type": "iot_event", "device_id": "sensor-01", "payload_summary": "motion",
	})
	var enc crypto.Envelope
	enc.Encrypt(key, plain)
	body, _ := json.Marshal(map[string]interface{}{"encrypted": enc})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/beacon", bytes.NewReader(body))
	req.Header.Set("X-Session-Key-Id", sessionID)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	srv.db.Audit(context.Background(), "agent:"+agentID, "iot_event", "iot", agentID, map[string]interface{}{
		"device_id": "sensor-01", "payload_summary": "motion", "gateway_id": agentID,
	})
	token := loginToken(t, srv)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var out map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &out)
	events, _ := out["events"].([]interface{})
	require.NotEmpty(t, events)
}

func TestAPI008_FilterEventsByGateway(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	ctx := context.Background()
	srv.db.Audit(ctx, "agent:gw1", "iot_event", "iot", "gw1", map[string]interface{}{"gateway_id": "gw1", "payload_summary": "a"})
	srv.db.Audit(ctx, "agent:gw2", "iot_event", "iot", "gw2", map[string]interface{}{"gateway_id": "gw2", "payload_summary": "b"})
	token := loginToken(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?gateway_id=gw1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var out map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &out)
	events, _ := out["events"].([]interface{})
	require.Len(t, events, 1)
}

func TestE2E001_AgentWhoami(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	agentID, sessionID, key := doHandshake(t, srv)
	token := loginToken(t, srv)
	taskBody, _ := json.Marshal(map[string]interface{}{
		"agent_id":     agentID,
		"command_type": "shell",
		"payload":      map[string]interface{}{"argv": []string{"whoami"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	var taskResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &taskResp)
	taskID := taskResp["task_id"]

	// beacon
	beaconPlain, _ := json.Marshal(map[string]interface{}{"type": "beacon", "agent_id": agentID})
	var enc crypto.Envelope
	enc.Encrypt(key, beaconPlain)
	b, _ := json.Marshal(map[string]interface{}{"encrypted": enc})
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/beacon", bytes.NewReader(b))
	reqB.Header.Set("X-Session-Key-Id", sessionID)
	wB := httptest.NewRecorder()
	srv.Router().ServeHTTP(wB, reqB)
	var out map[string]interface{}
	json.Unmarshal(wB.Body.Bytes(), &out)
	encRaw, _ := json.Marshal(out["encrypted"])
	var envelope crypto.Envelope
	json.Unmarshal(encRaw, &envelope)
	taskPlain, err := envelope.Decrypt(key)
	require.NoError(t, err)
	var taskMsg map[string]interface{}
	json.Unmarshal(taskPlain, &taskMsg)
	require.Equal(t, "task", taskMsg["type"])

	payload, _ := taskMsg["payload"].(map[string]interface{})
	res := ExecuteTaskOnAgent(context.Background(), srv.sim, "shell", payload)
	resultPlain, _ := json.Marshal(map[string]interface{}{
		"type":      "task_result",
		"task_id":   taskID,
		"exit_code": res.ExitCode,
		"stdout":    res.Stdout,
		"stderr":    res.Stderr,
	})
	var resEnc crypto.Envelope
	resEnc.Encrypt(key, resultPlain)
	resBody, _ := json.Marshal(map[string]interface{}{"encrypted": resEnc})
	reqR := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/task_result", bytes.NewReader(resBody))
	reqR.Header.Set("X-Session-Key-Id", sessionID)
	wR := httptest.NewRecorder()
	srv.Router().ServeHTTP(wR, reqR)

	task, err := srv.db.GetTask(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.NotEmpty(t, task.Stdout)
}

func TestIOT001_GatewayHandshake(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/handshake", bytes.NewReader([]byte(`{"step":"challenge_request"}`)))
	w1 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w1, req1)
	var ch map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &ch)
	nonce := ch["nonce"].(string)
	priv, _ := crypto.GenerateECDSAKeypair()
	agentECDH, _ := crypto.GenerateECDHKeypair()
	sig, _ := crypto.SignNonce(priv, nonce)
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&agentECDH.PublicKey)
	body := map[string]interface{}{
		"step":            "challenge_response",
		"nonce":           nonce,
		"agent_ecdsa_pub": crypto.ECDSAPublicKeyHex(priv),
		"agent_ecdh_pub":  base64.StdEncoding.EncodeToString(pubBytes),
		"signature":       sig,
		"hostname":        "gateway",
		"os":              "linux-arm64",
		"agent_role":      "iot_gateway",
		"gateway_id":      "gw-res-01",
		"timestamp":       time.Now().Unix(),
	}
	b, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/agents/handshake", bytes.NewReader(b))
	w2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w2, req2)
	var complete map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &complete)
	agentID := complete["agent_id"].(string)
	a, err := srv.db.GetAgent(context.Background(), agentID)
	require.NoError(t, err)
	require.Equal(t, "iot_gateway", a.AgentRole)
}

func TestIOT006_UnlockLock(t *testing.T) {
	s := sim.NewSimulator()
	res, err := s.LockCommand("lock-main", "unlock", 5)
	require.NoError(t, err)
	require.Equal(t, "unlocked", res["state"])
}

func TestE2EINTEG001_FullThreeLayerFlow(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	agentID, sessionID, key := doHandshake(t, srv)
	token := loginToken(t, srv)

	srv.db.Audit(context.Background(), "agent:"+agentID, "iot_event", "iot", agentID, map[string]interface{}{
		"device_id": "sensor-motion-01", "payload_summary": "motion zone-1", "gateway_id": agentID,
	})
	srv.chain.IndexConfigUpdated("0xabc", 30, 1, 42018)
	srv.db.UpsertChainConfig(context.Background(), "0xabc", `["https://localhost:8443"]`, 30, 1, 42018)

	plain, _ := json.Marshal(map[string]interface{}{"type": "beacon", "agent_id": agentID})
	var enc crypto.Envelope
	enc.Encrypt(key, plain)
	b, _ := json.Marshal(map[string]interface{}{"encrypted": enc})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID+"/beacon", bytes.NewReader(b))
	req.Header.Set("X-Session-Key-Id", sessionID)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	eventsReq := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	eventsReq.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w2, eventsReq)
	require.Equal(t, http.StatusOK, w2.Code)

	chainReq := httptest.NewRequest(http.MethodGet, "/api/v1/chain/status", nil)
	chainReq.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(w3, chainReq)
	require.Equal(t, http.StatusOK, w3.Code)

	a, err := srv.db.GetAgent(context.Background(), agentID)
	require.NoError(t, err)
	require.Equal(t, "active", a.Status)
}
