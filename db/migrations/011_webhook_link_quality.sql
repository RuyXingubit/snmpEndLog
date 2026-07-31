-- ============================================
-- Migration 011: Webhook Link Quality
-- ============================================

-- Adiciona a coluna para armazenar o token de webhook do dispositivo
ALTER TABLE devices ADD COLUMN IF NOT EXISTS webhook_token VARCHAR(64) UNIQUE;

-- Cria a tabela baseada em séries temporais para a qualidade do link externo
CREATE TABLE IF NOT EXISTS metric_link_quality (
    time            TIMESTAMPTZ NOT NULL,
    device_id       INTEGER NOT NULL,
    target_ip       VARCHAR(64) NOT NULL,
    rtt_min         DOUBLE PRECISION,
    rtt_avg         DOUBLE PRECISION,
    rtt_max         DOUBLE PRECISION,
    packet_loss     DOUBLE PRECISION NOT NULL
);

-- Transforma em hypertable do TimescaleDB
SELECT create_hypertable('metric_link_quality', 'time', if_not_exists => TRUE);

-- Cria índices para facilitar a busca por dispositivo e destino
CREATE INDEX IF NOT EXISTS idx_metric_link_quality_device ON metric_link_quality (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_metric_link_quality_target ON metric_link_quality (device_id, target_ip, time DESC);

-- Define a retenção de dados (ex: 30 dias)
SELECT add_retention_policy('metric_link_quality', INTERVAL '30 days', if_not_exists => TRUE);
