import { z } from "zod";

import { SsoMaxNameLength } from "@/ee/models/Sso";

export const SamlCertificateUse = { Sp: "sp", SpPrevious: "sp_previous", Idp: "idp" } as const;

export type SamlCertificateUse = (typeof SamlCertificateUse)[keyof typeof SamlCertificateUse];

export const SamlCertificateSchema = z.object({
  use: z.enum([SamlCertificateUse.Sp, SamlCertificateUse.SpPrevious, SamlCertificateUse.Idp]),
  not_after: z.string(),
  expires_soon: z.boolean(),
});

export type SamlCertificate = z.infer<typeof SamlCertificateSchema>;

// What the Owner reads back; Nexul's private key is never sent, only the certificates and the values to register.
export const SamlSettingsSchema = z.object({
  unlocked: z.boolean(),
  configured: z.boolean(),
  active: z.boolean(),
  metadata_url: z.string(),
  metadata_xml_set: z.boolean(),
  idp_entity_id: z.string(),
  idp_sso_url: z.string(),
  idp_certificates: z.number(),
  email_attribute: z.string(),
  name_attribute: z.string(),
  display_name: z.string(),
  join_role_id: z.string(),
  only: z.boolean(),
  sp: z.object({ entity_id: z.string(), acs_url: z.string(), metadata_url: z.string(), certificate: z.string(), previous_certificate: z.string() }),
  certificates: z.array(SamlCertificateSchema),
  metadata_refreshed_at: z.string().optional(),
  metadata_refresh_error: z.string(),
});

export type SamlSettings = z.infer<typeof SamlSettingsSchema>;

// A SAML provider is saved once its metadata named an identity provider.
export const samlSaved = (saml: SamlSettings): boolean => saml.idp_entity_id !== "";

export const MetadataSource = { Url: "url", Xml: "xml" } as const;

export type MetadataSource = (typeof MetadataSource)[keyof typeof MetadataSource];

// Uploaded metadata is never sent back, so an edit that keeps it leaves the XML blank.
export const samlFormSchema = (keepsXml: boolean) =>
  z
    .object({
      source: z.enum([MetadataSource.Url, MetadataSource.Xml]),
      metadata_url: z.string().trim(),
      metadata_xml: z.string(),
      email_attribute: z.string().trim().min(1, "Name the attribute the email is read from"),
      name_attribute: z.string().trim(),
      display_name: z.string().trim().min(1, "Enter the name the button shows").max(SsoMaxNameLength, `At most ${SsoMaxNameLength} characters`),
      join_role_id: z.string().min(1, "Choose the role new people get"),
    })
    .superRefine((v, ctx) => {
      if (v.source === MetadataSource.Url && v.metadata_url === "") ctx.addIssue({ code: "custom", path: ["metadata_url"], message: "Enter the metadata URL" });
      if (v.source === MetadataSource.Xml && v.metadata_xml.trim() === "" && !keepsXml)
        ctx.addIssue({ code: "custom", path: ["metadata_xml"], message: "Paste the metadata, or choose its file" });
    });

export type SamlFormData = z.infer<ReturnType<typeof samlFormSchema>>;

// The body the server takes: one metadata source, and no XML when an edit keeps the uploaded one.
export const samlBody = ({ source, metadata_url, metadata_xml, ...rest }: Record<string, string>): Record<string, string> => {
  if (source === MetadataSource.Url) return { ...rest, metadata_url: metadata_url ?? "" };
  if ((metadata_xml ?? "").trim() === "") return rest;
  return { ...rest, metadata_xml: metadata_xml ?? "" };
};

export const samlChecks = [
  { key: "metadata", label: "The metadata names a provider and a signing certificate", why: "Read from the URL now, or from the pasted XML." },
  { key: "email", label: "The email attribute is named", why: "A first sign-in is matched and made by the address in it." },
];
