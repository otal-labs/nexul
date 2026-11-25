-- name: GetSAMLSettings :one
SELECT * FROM saml_settings WHERE id = 1;

-- name: SetSAMLSettings :exec
INSERT INTO saml_settings (id, metadata_url, metadata_xml, idp_entity_id, idp_sso_url, idp_certificates, email_attribute,
    name_attribute, display_name, join_role_id, saml_only, sp_key, sp_certificate, updated_by, updated_at)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET metadata_url = excluded.metadata_url, metadata_xml = excluded.metadata_xml,
    idp_entity_id = excluded.idp_entity_id, idp_sso_url = excluded.idp_sso_url, idp_certificates = excluded.idp_certificates,
    email_attribute = excluded.email_attribute, name_attribute = excluded.name_attribute, display_name = excluded.display_name,
    join_role_id = excluded.join_role_id, saml_only = excluded.saml_only, sp_key = excluded.sp_key,
    sp_certificate = excluded.sp_certificate, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: DeleteSAMLSettings :exec
DELETE FROM saml_settings WHERE id = 1;
