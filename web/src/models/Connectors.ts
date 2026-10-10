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

// A passing check's answer; detail is a line for its row, such as the domains a token can edit, and absent on a 204.
export interface CheckResult {
  detail?: string;
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
  // True once the App's private key is stored: Nexul then reads GitHub as the App, not as the connected account.
  private_key_set?: boolean;
}

// Only GitHub Apps carry a slug and a self-hosted base URL; every other OAuth app is just client ID + secret.
// Editing a registered app may leave the secret blank, which keeps the stored one.
export const connectorAppConfigFormSchema = (githubApp: boolean, secretRequired = true) =>
  z.object({
    client_id: z.string().trim().min(1, "Client ID is required"),
    client_secret: secretRequired ? z.string().trim().min(1, "Client secret is required") : z.string().trim(),
    base_url: z.string().trim(),
    app_slug: githubApp
      ? z
          .string()
          .trim()
          .min(1, "App slug is required")
          .refine((v) => !/^\d+$/.test(v), "That's the numeric App ID. Enter the slug from your App's URL, github.com/apps/<slug>.")
      : z.string().trim(),
  });

// Where a GitHub App's public page lives: github.com, or the Enterprise server's base URL.
export const githubAppURL = (app: AppConfigStatus): string =>
  `${(app.base_url || "https://github.com").replace(/\/+$/, "")}/apps/${app.app_slug ?? ""}`;

// GitHub's own picker for the account or organisation to install the App on, and which of its repositories.
export const githubAppInstallURL = (app: AppConfigStatus): string => `${githubAppURL(app)}/installations/new`;

export type ConnectorAppConfigFormData = z.infer<ReturnType<typeof connectorAppConfigFormSchema>>;

// A GitHub App private key is the whole .pem GitHub downloads; the server parses and checks it with GitHub.
export const privateKeyFormSchema = z.object({
  private_key: z.string().trim().min(1, "Paste the private key"),
});

export type PrivateKeyFormData = z.infer<typeof privateKeyFormSchema>;
