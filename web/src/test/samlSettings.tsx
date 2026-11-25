import type { SamlSettings } from "@/ee/models/Saml";

// A saved SAML provider as the server reads it back, read by URL, with Nexul's one certificate and the provider's.
export const savedSaml = (over: Partial<SamlSettings> = {}): SamlSettings => ({
  unlocked: true,
  configured: true,
  active: true,
  metadata_url: "https://idp.example.com/metadata",
  metadata_xml_set: false,
  idp_entity_id: "https://idp.example.com",
  idp_sso_url: "https://idp.example.com/sso",
  idp_certificates: 1,
  email_attribute: "email",
  name_attribute: "",
  display_name: "Acme SSO",
  join_role_id: "role-member",
  only: false,
  sp: {
    entity_id: "https://nexul.example.com/auth/saml/metadata",
    acs_url: "https://nexul.example.com/auth/saml/acs",
    metadata_url: "https://nexul.example.com/auth/saml/metadata",
    certificate: "-----BEGIN CERTIFICATE-----",
    previous_certificate: "",
  },
  certificates: [
    { use: "sp", not_after: "2031-11-25T00:00:00Z", expires_soon: false },
    { use: "idp", not_after: "2030-01-01T00:00:00Z", expires_soon: false },
  ],
  metadata_refreshed_at: "2026-11-25T08:00:00Z",
  metadata_refresh_error: "",
  ...over,
});

// No SAML provider saved: only the values to register are known.
export const unsetSaml = (over: Partial<SamlSettings> = {}): SamlSettings =>
  savedSaml({
    configured: false,
    active: false,
    metadata_url: "",
    idp_entity_id: "",
    idp_sso_url: "",
    idp_certificates: 0,
    email_attribute: "",
    display_name: "",
    join_role_id: "",
    sp: { ...savedSaml().sp, certificate: "" },
    certificates: [],
    metadata_refreshed_at: undefined,
    ...over,
  });
