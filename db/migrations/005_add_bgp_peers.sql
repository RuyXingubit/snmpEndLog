-- ============================================
-- BGP Peers Table
-- ============================================

CREATE TABLE IF NOT EXISTS bgp_peers (
    id                  SERIAL PRIMARY KEY,
    device_id           INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    peer_addr           VARCHAR(64) NOT NULL,
    peer_as             BIGINT,
    state               INTEGER,
    admin_status        INTEGER,
    in_updates          BIGINT,
    out_updates         BIGINT,
    prefixes_received   BIGINT,
    prefixes_advertised BIGINT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(device_id, peer_addr)
);

CREATE INDEX IF NOT EXISTS idx_bgp_peers_device ON bgp_peers (device_id);
