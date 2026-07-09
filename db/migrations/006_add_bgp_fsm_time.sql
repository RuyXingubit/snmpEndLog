-- Add fsm_established_time to bgp_peers
ALTER TABLE bgp_peers ADD COLUMN fsm_established_time BIGINT;
