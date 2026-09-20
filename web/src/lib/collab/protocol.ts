// Wire protocol for collab sessions; shapes mirror internal/collab/protocol.go on the Go side.

export interface StoredUpdate {
  seq: number;
  kind: "update" | "snapshot";
  actor_id: string;
  payload: string;
  created_at?: string;
}

export interface PresenceMsg {
  client_id: number;
  payload: string;
}

export interface InitFrame {
  type: "init";
  seq: number;
  snapshot: StoredUpdate | null;
  updates: StoredUpdate[];
  presence: PresenceMsg[];
}

export interface RelayUpdateFrame {
  type: "update";
  from: string;
  seq: number;
  payload: string;
}

export interface RelayPresenceFrame {
  type: "presence";
  from: string;
  client_id: number;
  payload: string;
}

export interface RelayCommitFrame {
  type: "commit";
  from: string;
  seq: number;
  /** Present on a rename: peers hold the title in plain state (not the CRDT), so it rides the relay. */
  title?: string;
}

export interface LeaveFrame {
  type: "leave";
  client_id: number;
}

export type ServerFrame = InitFrame | RelayUpdateFrame | RelayPresenceFrame | RelayCommitFrame | LeaveFrame;

const isObject = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null;

const asString = (v: unknown): string => (typeof v === "string" ? v : "");

const parseInitFrame = (data: Record<string, unknown>): InitFrame => ({
  type: "init",
  seq: Number(data.seq ?? 0),
  snapshot: data.snapshot ? (data.snapshot as StoredUpdate) : null,
  updates: Array.isArray(data.updates) ? (data.updates as StoredUpdate[]) : [],
  presence: Array.isArray(data.presence) ? (data.presence as PresenceMsg[]) : [],
});

const parseUpdateFrame = (data: Record<string, unknown>): RelayUpdateFrame => ({
  type: "update",
  from: asString(data.from),
  seq: Number(data.seq ?? 0),
  payload: asString(data.payload),
});

const parsePresenceFrame = (data: Record<string, unknown>): RelayPresenceFrame => ({
  type: "presence",
  from: asString(data.from),
  client_id: Number(data.client_id ?? 0),
  payload: asString(data.payload),
});

const parseCommitFrame = (data: Record<string, unknown>): RelayCommitFrame => ({
  type: "commit",
  from: asString(data.from),
  seq: Number(data.seq ?? 0),
  ...(data.title ? { title: asString(data.title) } : {}),
});

const parseLeaveFrame = (data: Record<string, unknown>): LeaveFrame => ({
  type: "leave",
  client_id: Number(data.client_id ?? 0),
});

// Malformed frames throw so the provider can drop them with a log instead of killing the session.
export const parseServerFrame = (raw: string): ServerFrame => {
  const data = JSON.parse(raw) as unknown;
  if (!isObject(data)) throw new Error("collab frame is not an object");
  switch (data.type) {
    case "init":
      return parseInitFrame(data);
    case "update":
      return parseUpdateFrame(data);
    case "presence":
      return parsePresenceFrame(data);
    case "commit":
      return parseCommitFrame(data);
    case "leave":
      return parseLeaveFrame(data);
    default:
      throw new Error(`unknown collab frame type ${String(data.type)}`);
  }
};
