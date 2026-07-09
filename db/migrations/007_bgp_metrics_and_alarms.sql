-- ============================================
-- Migration 007: BGP Metrics and Alarms
-- ============================================

-- BGP Metrics Hypertable
CREATE TABLE IF NOT EXISTS metric_bgp (
    time            TIMESTAMPTZ NOT NULL,
    device_id       INTEGER NOT NULL,
    peer_addr       VARCHAR(64) NOT NULL,
    state           INTEGER,
    uptime          BIGINT
);

SELECT create_hypertable('metric_bgp', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metric_bgp_device ON metric_bgp (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_metric_bgp_peer ON metric_bgp (device_id, peer_addr, time DESC);

SELECT add_retention_policy('metric_bgp', INTERVAL '30 days', if_not_exists => TRUE);

-- Alarms Table
CREATE TABLE IF NOT EXISTS alarms (
    id              SERIAL PRIMARY KEY,
    device_id       INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    entity_type     VARCHAR(32) NOT NULL, -- e.g., 'interface', 'bgp_peer'
    entity_id       VARCHAR(255) NOT NULL, -- e.g., if_index or peer_addr
    name            VARCHAR(255) NOT NULL, -- Short description
    severity        VARCHAR(16) NOT NULL DEFAULT 'critical', -- 'critical', 'warning', 'info'
    status          VARCHAR(16) NOT NULL DEFAULT 'active', -- 'active', 'resolved'
    message         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alarms_device ON alarms (device_id);
CREATE INDEX IF NOT EXISTS idx_alarms_status ON alarms (status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_alarms_unique_active ON alarms (device_id, entity_type, entity_id) WHERE status = 'active';
