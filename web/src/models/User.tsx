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
  provider: Provider;
  provider_user_id: string;
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
  // A free-text template with {ticket.Field} placeholders; any user can read it, only managers can change it.
  mention_chip_template: string;
  // Shared by the connection-token flow and the pairing settings snippet; empty until an instance URL is set.
  mcp_url?: string;
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

export const AddMemberFormSchema = z.object({
  login: z.string().trim().min(1, "GitHub username or Google/Discord email is required"),
});

export type AddMemberFormData = z.infer<typeof AddMemberFormSchema>;

// Only GitHub exposes a user-search API, so Google/Discord emails never get typeahead suggestions.
export interface LoginMatch {
  login: string;
  avatar_url: string;
}

export const OAuthProviderFormSchema = z
  .object({
    client_id: z.string().trim(),
    client_secret: z.string().trim(),
  })
  .refine((v) => (v.client_id === "") === (v.client_secret === ""), {
    message: "Enter both the client ID and secret, or clear both to turn this sign-in off",
    path: ["client_secret"],
  });

export type OAuthProviderFormData = z.infer<typeof OAuthProviderFormSchema>;

export const PATNameFormSchema = z.object({
  name: z.string().trim().min(1, "Token name is required"),
});

export type PATNameFormData = z.infer<typeof PATNameFormSchema>;
