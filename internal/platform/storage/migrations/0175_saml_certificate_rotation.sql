-- Neither side's certificate expiry locks anyone out (ADR 0189): sp_previous_certificate is Nexul's certificate a
-- rotation replaced, published beside the new one until the Owner retires it; refreshed_at and refresh_error record the
-- last read of a metadata URL, whose failure keeps the certificates read before it.
ALTER TABLE saml_settings ADD COLUMN sp_previous_certificate TEXT NOT NULL DEFAULT '';
ALTER TABLE saml_settings ADD COLUMN refreshed_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE saml_settings ADD COLUMN refresh_error TEXT NOT NULL DEFAULT '';
