-- A paired computer's own tunnel (ADR 0062); every column is empty for a computer paired by URL.
ALTER TABLE pairing_computers ADD COLUMN tunnel_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pairing_computers ADD COLUMN tunnel_hostname TEXT NOT NULL DEFAULT '';
ALTER TABLE pairing_computers ADD COLUMN tunnel_zone_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pairing_computers ADD COLUMN tunnel_record_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pairing_computers ADD COLUMN tunnel_access_app_id TEXT NOT NULL DEFAULT '';

-- Serves PairingComputerTunnelHostnameExists, which decides whether a request may carry the Access headers.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pairing_computers_tunnel_hostname
    ON pairing_computers(tunnel_hostname) WHERE tunnel_hostname != '';
