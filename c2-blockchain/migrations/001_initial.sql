CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);
INSERT OR IGNORE INTO schema_version (version) VALUES (1);

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    ecdsa_pub TEXT NOT NULL UNIQUE,
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    agent_role TEXT NOT NULL DEFAULT 'generic',
    residential_unit TEXT,
    gateway_id TEXT,
    first_seen DATETIME NOT NULL,
    last_beacon DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_last_beacon ON agents(last_beacon);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    session_key_enc BLOB NOT NULL,
    established_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    status TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_agent_id ON sessions(agent_id);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    command_type TEXT NOT NULL,
    payload_enc BLOB,
    status TEXT NOT NULL,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    created_by TEXT NOT NULL,
    chain_tx_hash TEXT,
    created_at DATETIME NOT NULL,
    completed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_tasks_agent_status ON tasks(agent_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT,
    metadata_json TEXT,
    timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor);

CREATE TABLE IF NOT EXISTS chain_config_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_hash TEXT NOT NULL,
    endpoints_json TEXT NOT NULL,
    beacon_interval_sec INTEGER NOT NULL,
    version INTEGER NOT NULL UNIQUE,
    block_number INTEGER NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS iot_devices (
    id TEXT PRIMARY KEY,
    gateway_agent_id TEXT NOT NULL REFERENCES agents(id),
    device_type TEXT NOT NULL,
    zone TEXT,
    pub_key_hash TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    lock_state TEXT,
    last_event_at DATETIME
);
