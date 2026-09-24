// Matches internal/connectors' model.go and usecase.go JSON exactly.

import { z } from "zod";

// A manual-credential connector's form field definition, never a value; secret renders as a password input.
export interface CredentialField {
  key: string;
  label: string;
  secret: boolean;
  hint?: string;
}

export interface Connector {
  id: string;
  name: string;
  description: string;
  category: string;
  // A plain lowercase identifier ("github", "cloudflare") the frontend maps to an actual icon.
  icon: string;
  docs_url?: string;
  // Set only for manual-credential connectors; its presence makes ConnectorCard render a field form.
  manual?: CredentialField[];
  // Named permissions the Connect dialog verifies one row each, in parallel; absent means a single pass/fail.
  checks?: CredentialCheck[];
}

export interface CredentialCheck {
  key: string;
  label: string;
  // Shown under the permission so a red row also explains what Nexul needs it for.
  why?: string;
  // An advisory check that fails shows a warning but never blocks Confirm.
  advisory?: boolean;
}

export interface CredentialStatus {
  configured: boolean;
  connected_by?: string;
  connected_at?: string;
  expires_at?: string;
}

export interface ConnectorStatus {
  connector: Connector;
  status: CredentialStatus;
  // True only with a real OAuth implementation wired server-side; false renders as "coming soon".
  available: boolean;
  // True once an owner stored the OAuth app registration; false renders "Set up app" in place of Connect.
  app_configured: boolean;
}

// The instance-wide OAuth app registration, never the per-connection user token ConnectorStatus tracks.
export interface AppConfigStatus {
  configured: boolean;
  client_id?: string;
  base_url?: string;
  app_slug?: string;
}

// Only GitHub Apps carry a slug and a self-hosted base URL; every other OAuth app is just client ID + secret.
export const connectorAppConfigFormSchema = (githubApp: boolean) =>
  z.object({
    client_id: z.string().trim().min(1, "Client ID is required"),
    client_secret: z.string().trim().min(1, "Client secret is required"),
    base_url: z.string().trim(),
    app_slug: githubApp
      ? z
          .string()
          .trim()
          .min(1, "App slug is required")
          .refine((v) => !/^\d+$/.test(v), "That's the numeric App ID — enter the slug from your app's URL (github.com/apps/<slug>)")
      : z.string().trim(),
  });

export type ConnectorAppConfigFormData = z.infer<ReturnType<typeof connectorAppConfigFormSchema>>;
