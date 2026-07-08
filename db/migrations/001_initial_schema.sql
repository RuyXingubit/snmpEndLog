-- ============================================
-- Migration 001: Initial Schema
-- snmpEndLog — PostgreSQL + TimescaleDB
-- ============================================

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ============================================
-- Users (Dashboard Authentication)
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    username        VARCHAR(64) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(16) NOT NULL DEFAULT 'viewer',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- ============================================
-- Devices (Monitored Equipment)
-- ============================================
CREATE TABLE IF NOT EXISTS devices (
    id              SERIAL PRIMARY KEY,
    hostname        VARCHAR(255) NOT NULL,
    ip_address      INET NOT NULL UNIQUE,
    snmp_version    VARCHAR(4) NOT NULL DEFAULT 'v2c',

    -- SNMPv2c
    community       VARCHAR(255),

    -- SNMPv3
    snmpv3_user         VARCHAR(255),
    snmpv3_auth_proto   VARCHAR(16),
    snmpv3_auth_pass    VARCHAR(255),
    snmpv3_priv_proto   VARCHAR(16),
    snmpv3_priv_pass    VARCHAR(255),
    snmpv3_sec_level    VARCHAR(24) DEFAULT 'authPriv',

    poll_interval   INTEGER NOT NULL DEFAULT 300,
    ping_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    sys_descr       TEXT,
    sys_name        VARCHAR(255),
    sys_location    VARCHAR(255),
    sys_contact     VARCHAR(255),
    sys_uptime      BIGINT,
    last_polled_at  TIMESTAMPTZ,
    last_seen_at    TIMESTAMPTZ,
    status          VARCHAR(16) NOT NULL DEFAULT 'unknown',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices (ip_address);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices (status);

-- ============================================
-- Interfaces (Discovered per device)
-- ============================================
CREATE TABLE IF NOT EXISTS interfaces (
    id              SERIAL PRIMARY KEY,
    device_id       INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    if_index        INTEGER NOT NULL,
    if_descr        VARCHAR(255),
    if_alias        VARCHAR(255),
    if_type         INTEGER,
    if_speed        BIGINT,
    if_hc_speed     BIGINT,
    if_admin_status INTEGER DEFAULT 1,
    if_oper_status  INTEGER DEFAULT 1,
    if_phys_address VARCHAR(17),
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(device_id, if_index)
);

CREATE INDEX IF NOT EXISTS idx_interfaces_device ON interfaces (device_id);

-- ============================================
-- Metric: Interface Traffic (Hypertable)
-- ============================================
CREATE TABLE IF NOT EXISTS metric_traffic (
    time            TIMESTAMPTZ NOT NULL,
    device_id       INTEGER NOT NULL,
    if_index        INTEGER NOT NULL,
    in_octets       BIGINT,
    out_octets      BIGINT,
    in_bps          DOUBLE PRECISION,
    out_bps         DOUBLE PRECISION,
    in_errors       BIGINT DEFAULT 0,
    out_errors      BIGINT DEFAULT 0
);

SELECT create_hypertable('metric_traffic', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metric_traffic_device ON metric_traffic (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_metric_traffic_if ON metric_traffic (device_id, if_index, time DESC);

-- ============================================
-- Metric: System (CPU, Memory, Uptime) (Hypertable)
-- ============================================
CREATE TABLE IF NOT EXISTS metric_system (
    time            TIMESTAMPTZ NOT NULL,
    device_id       INTEGER NOT NULL,
    cpu_percent     DOUBLE PRECISION,
    memory_percent  DOUBLE PRECISION,
    memory_used     BIGINT,
    memory_total    BIGINT,
    uptime          BIGINT
);

SELECT create_hypertable('metric_system', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metric_system_device ON metric_system (device_id, time DESC);

-- ============================================
-- Metric: Ping / ICMP (Hypertable)
-- ============================================
CREATE TABLE IF NOT EXISTS metric_ping (
    time            TIMESTAMPTZ NOT NULL,
    device_id       INTEGER NOT NULL,
    rtt_min         DOUBLE PRECISION,
    rtt_avg         DOUBLE PRECISION,
    rtt_max         DOUBLE PRECISION,
    packet_loss     DOUBLE PRECISION,
    is_reachable    BOOLEAN NOT NULL DEFAULT TRUE
);

SELECT create_hypertable('metric_ping', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_metric_ping_device ON metric_ping (device_id, time DESC);

-- ============================================
-- Logs (Syslog Messages) (Hypertable)
-- ============================================
CREATE TABLE IF NOT EXISTS logs (
    time            TIMESTAMPTZ NOT NULL,
    host            VARCHAR(255) NOT NULL,
    device_id       INTEGER,
    facility        SMALLINT,
    severity        SMALLINT,
    facility_name   VARCHAR(32),
    severity_name   VARCHAR(16),
    app_name        VARCHAR(255),
    message         TEXT NOT NULL,
    raw             TEXT
);

SELECT create_hypertable('logs', 'time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_logs_host ON logs (host, time DESC);
CREATE INDEX IF NOT EXISTS idx_logs_severity ON logs (severity, time DESC);
CREATE INDEX IF NOT EXISTS idx_logs_device ON logs (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_logs_message ON logs USING GIN (to_tsvector('simple', message));

-- ============================================
-- Retention Policies (TimescaleDB)
-- ============================================
SELECT add_retention_policy('metric_traffic', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('metric_system',  INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('metric_ping',    INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('logs',           INTERVAL '30 days', if_not_exists => TRUE);

-- ============================================
-- Continuous Aggregates (Hourly summaries)
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS metric_traffic_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    device_id,
    if_index,
    AVG(in_bps)  AS avg_in_bps,
    MAX(in_bps)  AS max_in_bps,
    AVG(out_bps) AS avg_out_bps,
    MAX(out_bps) AS max_out_bps,
    SUM(in_errors)  AS total_in_errors,
    SUM(out_errors) AS total_out_errors
FROM metric_traffic
GROUP BY bucket, device_id, if_index
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metric_traffic_hourly',
    start_offset    => INTERVAL '4 hours',
    end_offset      => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists   => TRUE
);

CREATE MATERIALIZED VIEW IF NOT EXISTS metric_system_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    device_id,
    AVG(cpu_percent)    AS avg_cpu,
    MAX(cpu_percent)    AS max_cpu,
    AVG(memory_percent) AS avg_memory,
    MAX(memory_percent) AS max_memory
FROM metric_system
GROUP BY bucket, device_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metric_system_hourly',
    start_offset    => INTERVAL '4 hours',
    end_offset      => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists   => TRUE
);

-- Retain hourly aggregates for 90 days
SELECT add_retention_policy('metric_traffic_hourly', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('metric_system_hourly',  INTERVAL '90 days', if_not_exists => TRUE);
