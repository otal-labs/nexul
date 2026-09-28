export type SessionClient = "browser" | "desktop" | "phone";

export interface DeviceSession {
  id: string;
  client: SessionClient;
  label: string;
  ip: string;
  last_used_at: string;
  current: boolean;
}
