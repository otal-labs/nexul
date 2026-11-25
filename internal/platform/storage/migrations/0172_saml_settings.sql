-- The instance's single sign-on provider when it speaks SAML (ADR 0189): a singleton row beside sso_settings, which the
-- use-case keeps to one of the two. What was read from the identity provider's metadata is stored so a sign-in never
-- parses it; idp_certificates is a JSON array of base64 DER. sp_key is Nexul's own private key, sealed.
CREATE TABLE IF NOT EXISTS saml_settings (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    metadata_url     TEXT    NOT NULL DEFAULT '',
    metadata_xml     TEXT    NOT NULL DEFAULT '',
    idp_entity_id    TEXT    NOT NULL,
    idp_sso_url      TEXT    NOT NULL,
    idp_certificates TEXT    NOT NULL,
    email_attribute  TEXT    NOT NULL,
    name_attribute   TEXT    NOT NULL DEFAULT '',
    display_name     TEXT    NOT NULL,
    join_role_id     TEXT    NOT NULL,
    saml_only        INTEGER NOT NULL DEFAULT 0,
    sp_key           TEXT    NOT NULL,
    sp_certificate   TEXT    NOT NULL,
    updated_by       TEXT    NOT NULL DEFAULT '',
    updated_at       INTEGER NOT NULL
);
