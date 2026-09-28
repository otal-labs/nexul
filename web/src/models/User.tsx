import { z } from "zod";

export const Provider = {
  GitHub: "github",
  Google: "google",
  Discord: "discord",
} as const;

// The optional, owner-configurable sign-in providers (ADR 0040); GitHub is set up by bootstrap, not here.
export type OptionalProvider = "google" | "discord";

export type Provider = (typeof Provider)[keyof typeof Provider];

export interface User {
  id: string;
  login: string;
  name: string;
  avatar_url: string;
  can_create_workspace: boolean;
  first_login_done: boolean;
  created_at: string;
  // The optional manual profile override; omitted from the JSON entirely when unset, hence optional not nullable.
  display_name?: string;
  avatar_override_url?: string;
}

// One provider account attached to the user; the provider's own id never leaves the server.
export interface Identity {
  provider: Provider;
  login: string;
  name: string;
  avatar_url: string;
  created_at: string;
}

export const providerLabel: Record<Provider, string> = { github: "GitHub", google: "Google", discord: "Discord" };

// The one rule for which avatar to show: the manual override if set, else the provider-sourced one.
export const effectiveAvatar = (user: User): string => user.avatar_override_url || user.avatar_url;

export interface MeResponse {
  user: User;
  needs_owner_wizard: boolean;
  needs_first_login_wizard: boolean;
}

export interface InstanceSettings {
  instance_url: string;
  settings_version: number;
  oauth_callback: string;
  // The client ID is public, the secret is never sent — only whether one is stored.
  google_oauth_client_id?: string;
  google_oauth_callback?: string;
  google_oauth_configured?: boolean;
  discord_oauth_client_id?: string;
  discord_oauth_callback?: string;
  discord_oauth_configured?: boolean;
}

export interface ConnectionToken {
  token: string;
  instance_url: string;
  mcp_url?: string;
  settings_version: number;
  expires_at: string;
}

export interface PersonalAccessToken {
  id: string;
  user_id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string | null;
  revoked_at?: string | null;
  // Set when the token was minted for a paired computer's MCP connection.
  computer_id?: string;
}

export type SessionClient = "browser" | "desktop" | "phone";

// One signed-in device; `current` flags the session the listing request itself came in on.
export interface Session {
  id: string;
  user_id: string;
  client: SessionClient;
  platform: string;
  label: string;
  ip: string;
  created_at: string;
  last_active_at: string;
  expires_at: string;
  current: boolean;
}

export interface MintPATResponse {
  token: string;
  id: string;
  name: string;
  prefix: string;
  created_at: string;
}

export const InstanceURLFormSchema = z.object({
  instance_url: z.string().url("Enter a valid URL, e.g. https://deploy.example.com"),
});

export type InstanceURLFormData = z.infer<typeof InstanceURLFormSchema>;

export interface BootstrapStatus {
  configured: boolean;
  // True while no user has logged in yet: the stored GitHub App may be replaced from /setup.
  reconfigurable?: boolean;
  google_configured?: boolean;
  discord_configured?: boolean;
  // No user exists yet, so a setup pass can still unlock the first-run screens.
  setup_open?: boolean;
  // Stored by the domain step; empty until then.
  instance_url?: string;
  // Desktop install: first run keeps localhost and skips the domain step.
  local?: boolean;
}

export interface BootstrapResponse {
  instance_url: string;
  settings_version: number;
  client_id: string;
  configured: boolean;
}

export const InstanceBootstrapFormSchema = z.object({
  instance_url: z.string().url("Enter a valid URL, e.g. https://deploy.example.com"),
  client_id: z.string().trim().min(1, "Client ID is required"),
  client_secret: z.string().trim().min(1, "Client secret is required"),
  app_slug: z
    .string()
    .trim()
    .min(1, "App slug is required")
    .refine((v) => !/^\d+$/.test(v), "That's the numeric App ID — enter the slug from your app's URL (github.com/apps/<slug>)"),
});

export type InstanceBootstrapFormData = z.infer<typeof InstanceBootstrapFormSchema>;

// A first setup needs both values; editing an enabled provider may leave the secret blank to keep the stored one.
export const oauthProviderFormSchema = (editing: boolean) =>
  z
    .object({
      client_id: z.string().trim().min(1, "Client ID is required"),
      client_secret: z.string().trim(),
    })
    .refine((v) => editing || v.client_secret !== "", {
      message: "Client secret is required",
      path: ["client_secret"],
    });

export const OAuthProviderFormSchema = oauthProviderFormSchema(false);

// Per-provider copy only; the flow (enable with both values, edit, disable) is identical.
export const oauthProviderCopy: Record<
  OptionalProvider,
  { label: string; console: string; idPlaceholder: string; secretPlaceholder: string }
> = {
  google: {
    label: "Google",
    console: "Create an OAuth client (Web application) in Google Cloud Console",
    idPlaceholder: "1234567890-abc.apps.googleusercontent.com",
    secretPlaceholder: "GOCSPX-…",
  },
  discord: {
    label: "Discord",
    console: "Create an application in the Discord Developer Portal (OAuth2 → General)",
    idPlaceholder: "123456789012345678",
    secretPlaceholder: "Client secret from the OAuth2 page",
  },
};

export type OAuthProviderFormData = z.infer<typeof OAuthProviderFormSchema>;

export const PATNameFormSchema = z.object({
  name: z.string().trim().min(1, "Token name is required"),
});

export type PATNameFormData = z.infer<typeof PATNameFormSchema>;
