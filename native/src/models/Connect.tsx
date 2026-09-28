export interface About {
  product: string;
  version: string;
}

export interface ConnectDevice {
  model: string;
  os: string;
  app_version: string;
}

export interface ConnectExchange {
  token: string;
  server_version: string;
}

export interface ConnectLink {
  host: string;
  code: string;
}

export type ConnectResult = { kind: "connected" } | { kind: "refused"; host: string; version: string };
