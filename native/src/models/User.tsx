export interface User {
  id: string;
  login: string;
  name: string;
  avatar_url: string;
  display_name?: string;
  avatar_override_url?: string;
}

// The one rule for which avatar to show: the manual override if set, else the provider-sourced one.
export const effectiveAvatar = (user: User): string => user.avatar_override_url || user.avatar_url;

export interface MeResponse {
  user: User;
}

export const Provider = {
  GitHub: "github",
  Google: "google",
  Discord: "discord",
} as const;

export type Provider = (typeof Provider)[keyof typeof Provider];

export const providerLabel: Record<string, string> = { github: "GitHub", google: "Google", discord: "Discord" };

// One provider account attached to the user; the provider's own id never leaves the server.
export interface Identity {
  provider: Provider;
  login: string;
  name: string;
  // A Discord login is its email; the username is what to show.
  username: string;
}

export type SessionClient = "browser" | "desktop" | "phone";

// One signed-in device; `current` flags the session the listing request itself came in on.
export interface Session {
  id: string;
  client: SessionClient;
  platform: string;
  label: string;
  ip: string;
  last_active_at: string;
  current: boolean;
}
