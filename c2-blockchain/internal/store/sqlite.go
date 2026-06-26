package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	return s, nil
}

func (s *SQLite) Migrate(migrationPath string) error {
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(data))
	return err
}

func (s *SQLite) DB() *sql.DB { return s.db }

func (s *SQLite) Close() error { return s.db.Close() }

type Agent struct {
	ID              string
	ECDSAPub        string
	Hostname        string
	OS              string
	Status          string
	AgentRole       string
	ResidentialUnit string
	GatewayID       string
	FirstSeen       time.Time
	LastBeacon      *time.Time
}

func (s *SQLite) UpsertAgent(ctx context.Context, a Agent) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, ecdsa_pub, hostname, os, status, agent_role, residential_unit, gateway_id, first_seen, last_beacon, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET hostname=excluded.hostname, os=excluded.os, status=excluded.status,
			agent_role=excluded.agent_role, residential_unit=excluded.residential_unit, gateway_id=excluded.gateway_id,
			last_beacon=excluded.last_beacon, updated_at=excluded.updated_at
	`, a.ID, a.ECDSAPub, a.Hostname, a.OS, a.Status, a.AgentRole, a.ResidentialUnit, a.GatewayID, a.FirstSeen, a.LastBeacon, now, now)
	return err
}

func (s *SQLite) UpdateLastBeacon(ctx context.Context, agentID string, t time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET last_beacon=?, updated_at=? WHERE id=?`, t, t, agentID)
	return err
}

func (s *SQLite) ListAgents(ctx context.Context) ([]Agent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, ecdsa_pub, hostname, os, status, agent_role, residential_unit, gateway_id, first_seen, last_beacon FROM agents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		var lb sql.NullTime
		if err := rows.Scan(&a.ID, &a.ECDSAPub, &a.Hostname, &a.OS, &a.Status, &a.AgentRole, &a.ResidentialUnit, &a.GatewayID, &a.FirstSeen, &lb); err != nil {
			return nil, err
		}
		if lb.Valid {
			a.LastBeacon = &lb.Time
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *SQLite) GetAgent(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	var lb sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT id, ecdsa_pub, hostname, os, status, agent_role, residential_unit, gateway_id, first_seen, last_beacon FROM agents WHERE id=?`, id).
		Scan(&a.ID, &a.ECDSAPub, &a.Hostname, &a.OS, &a.Status, &a.AgentRole, &a.ResidentialUnit, &a.GatewayID, &a.FirstSeen, &lb)
	if err != nil {
		return nil, err
	}
	if lb.Valid {
		a.LastBeacon = &lb.Time
	}
	return &a, nil
}

type Session struct {
	ID           string
	AgentID      string
	SessionKey   []byte
	Established  time.Time
	ExpiresAt    time.Time
	Status       string
}

func (s *SQLite) CreateSession(ctx context.Context, sess Session, keyEnc []byte) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (id, agent_id, session_key_enc, established_at, expires_at, status) VALUES (?,?,?,?,?,?)`,
		sess.ID, sess.AgentID, keyEnc, sess.Established, sess.ExpiresAt, sess.Status)
	return err
}

func (s *SQLite) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	var keyEnc []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, agent_id, session_key_enc, established_at, expires_at, status FROM sessions WHERE id=? AND status='active'`, id).
		Scan(&sess.ID, &sess.AgentID, &keyEnc, &sess.Established, &sess.ExpiresAt, &sess.Status)
	if err != nil {
		return nil, err
	}
	sess.SessionKey = keyEnc
	return &sess, nil
}

func (s *SQLite) DecryptSessionKey(masterKey []byte, enc []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return enc, nil // lab: store raw if no master key
	}
	// simplified: keys stored as hex in lab when master set
	if len(enc) == 32 {
		return enc, nil
	}
	return enc, nil
}

type Task struct {
	ID          string
	AgentID     string
	CommandType string
	PayloadJSON string
	Status      string
	ExitCode    *int
	Stdout      string
	Stderr      string
	CreatedBy   string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (s *SQLite) CreateTask(ctx context.Context, t Task) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO tasks (id, agent_id, command_type, payload_enc, status, created_by, created_at) VALUES (?,?,?,?,?,?,?)`,
		t.ID, t.AgentID, t.CommandType, []byte(t.PayloadJSON), t.Status, t.CreatedBy, t.CreatedAt)
	return err
}

func (s *SQLite) GetTask(ctx context.Context, id string) (*Task, error) {
	var t Task
	var payload []byte
	var exit sql.NullInt64
	var completed sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT id, agent_id, command_type, payload_enc, status, exit_code, stdout, stderr, created_by, created_at, completed_at FROM tasks WHERE id=?`, id).
		Scan(&t.ID, &t.AgentID, &t.CommandType, &payload, &t.Status, &exit, &t.Stdout, &t.Stderr, &t.CreatedBy, &t.CreatedAt, &completed)
	if err != nil {
		return nil, err
	}
	t.PayloadJSON = string(payload)
	if exit.Valid {
		v := int(exit.Int64)
		t.ExitCode = &v
	}
	if completed.Valid {
		t.CompletedAt = &completed.Time
	}
	return &t, nil
}

func (s *SQLite) CompleteTask(ctx context.Context, id string, exitCode int, stdout, stderr string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status='completed', exit_code=?, stdout=?, stderr=?, completed_at=? WHERE id=?`,
		exitCode, stdout, stderr, now, id)
	return err
}

func (s *SQLite) Audit(ctx context.Context, actor, action, resource, resourceID string, meta map[string]interface{}) error {
	metaJSON, _ := json.Marshal(meta)
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_log (actor, action, resource, resource_id, metadata_json, timestamp) VALUES (?,?,?,?,?,?)`,
		actor, action, resource, resourceID, string(metaJSON), time.Now())
	return err
}

func (s *SQLite) ListIoTEvents(ctx context.Context, gatewayID string, limit, offset int) ([]map[string]interface{}, int, error) {
	q := `SELECT id, actor, action, metadata_json, timestamp FROM audit_log WHERE action IN ('iot_event','iot_telemetry','iot_command_result')`
	args := []interface{}{}
	if gatewayID != "" {
		q += ` AND json_extract(metadata_json, '$.gateway_id') = ?`
		args = append(args, gatewayID)
	}
	q += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var events []map[string]interface{}
	for rows.Next() {
		var id int
		var actor, action, metaJSON string
		var ts time.Time
		if err := rows.Scan(&id, &actor, &action, &metaJSON, &ts); err != nil {
			return nil, 0, err
		}
		ev := map[string]interface{}{
			"id":         fmt.Sprintf("%d", id),
			"event_type": action,
			"created_at": ts.Format(time.RFC3339),
		}
		if len(actor) > 6 && actor[:6] == "agent:" {
			ev["agent_id"] = actor[6:]
		}
		var meta map[string]interface{}
		json.Unmarshal([]byte(metaJSON), &meta)
		if meta != nil {
			for k, v := range meta {
				ev[k] = v
			}
		}
		events = append(events, ev)
	}
	countQ := `SELECT COUNT(*) FROM audit_log WHERE action IN ('iot_event','iot_telemetry','iot_command_result')`
	var total int
	if gatewayID != "" {
		s.db.QueryRowContext(ctx, countQ+` AND json_extract(metadata_json, '$.gateway_id') = ?`, gatewayID).Scan(&total)
	} else {
		s.db.QueryRowContext(ctx, countQ).Scan(&total)
	}
	return events, total, rows.Err()
}

func (s *SQLite) UpsertChainConfig(ctx context.Context, hash string, endpointsJSON string, beaconSec, version, block int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chain_config_cache (config_hash, endpoints_json, beacon_interval_sec, version, block_number, updated_at)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(version) DO UPDATE SET config_hash=excluded.config_hash, endpoints_json=excluded.endpoints_json,
			beacon_interval_sec=excluded.beacon_interval_sec, block_number=excluded.block_number, updated_at=excluded.updated_at
	`, hash, endpointsJSON, beaconSec, version, block, time.Now())
	return err
}

func (s *SQLite) LatestChainConfig(ctx context.Context) (hash string, endpointsJSON string, beaconSec, version, block int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT config_hash, endpoints_json, beacon_interval_sec, version, block_number FROM chain_config_cache ORDER BY version DESC LIMIT 1`).
		Scan(&hash, &endpointsJSON, &beaconSec, &version, &block)
	return
}

func (s *SQLite) UpsertIoTDevice(ctx context.Context, id, gatewayID, deviceType, zone, lockState string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO iot_devices (id, gateway_agent_id, device_type, zone, status, lock_state, last_event_at)
		VALUES (?,?,?,?, 'active', ?, ?)
		ON CONFLICT(id) DO UPDATE SET lock_state=excluded.lock_state, last_event_at=excluded.last_event_at
	`, id, gatewayID, deviceType, zone, lockState, time.Now())
	return err
}

func (s *SQLite) GetIoTDeviceState(ctx context.Context, id string) (deviceType, lockState string, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT device_type, lock_state FROM iot_devices WHERE id=?`, id).Scan(&deviceType, &lockState)
	return
}

func (s *SQLite) ListIoTDevices(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, gateway_agent_id, device_type, zone, status, lock_state, last_event_at
		FROM iot_devices ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, gw, dtype, zone, status, lockState string
		var lastEvent sql.NullTime
		if err := rows.Scan(&id, &gw, &dtype, &zone, &status, &lockState, &lastEvent); err != nil {
			return nil, err
		}
		m := map[string]interface{}{
			"device_id":   id,
			"gateway_id":  gw,
			"device_type": dtype,
			"zone":        zone,
			"status":      status,
			"lock_state":  lockState,
		}
		if lastEvent.Valid {
			m["last_event_at"] = lastEvent.Time.Format(time.RFC3339)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func MasterKeyFromHex(hexStr string) ([]byte, error) {
	if hexStr == "" {
		return nil, nil
	}
	return hex.DecodeString(hexStr)
}
