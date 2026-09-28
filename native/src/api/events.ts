export interface ServerFrame {
  topic: string;
  type: string;
  payload: unknown;
}

export interface LiveSocket {
  onopen: ((ev: unknown) => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  onclose: ((ev: unknown) => void) | null;
  onerror: ((ev: unknown) => void) | null;
  close: () => void;
}

export type LiveEventsHandler = (frame: ServerFrame) => void;

export interface LiveEventsClientOptions {
  wsFactory?: (url: string) => LiveSocket;
}

const MAX_BACKOFF_MS = 30_000;
const BASE_BACKOFF_MS = 500;
const JITTER_MS = 250;

export const buildLiveURL = (host: string, token: string): string =>
  `${host.replace(/^http/i, "ws")}/ws/events?token=${encodeURIComponent(token)}`;

export const parseFrame = (raw: string): ServerFrame => {
  const data = JSON.parse(raw) as unknown;
  if (typeof data !== "object" || data === null) throw new Error("frame is not an object");
  const { topic, type, payload } = data as Record<string, unknown>;
  if (typeof topic !== "string" || typeof type !== "string") throw new Error("frame missing topic/type");
  return { topic, type, payload };
};

export class LiveEventsClient {
  private socket: LiveSocket | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private attempt = 0;
  private closed = false;
  private readonly handlers = new Set<LiveEventsHandler>();
  private readonly wsFactory: (url: string) => LiveSocket;

  constructor(
    private readonly url: string,
    opts: LiveEventsClientOptions = {},
  ) {
    this.wsFactory = opts.wsFactory ?? ((u) => new WebSocket(u) as unknown as LiveSocket);
  }

  connect() {
    if (this.closed) return;
    const socket = this.wsFactory(this.url);
    this.socket = socket;
    socket.onopen = () => {
      this.attempt = 0;
    };
    socket.onmessage = (ev) => this.handleMessage(ev.data);
    socket.onerror = () => console.warn("ws connection error");
    socket.onclose = () => this.scheduleReconnect();
  }

  subscribe(handler: LiveEventsHandler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  close() {
    this.closed = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    const socket = this.socket;
    if (socket) {
      // Detach first, or the torn-down socket's close event schedules a reconnect.
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      socket.close();
    }
    this.socket = null;
    this.handlers.clear();
  }

  private handleMessage(raw: string) {
    try {
      const frame = parseFrame(raw);
      this.handlers.forEach((handler) => handler(frame));
    } catch {
      console.warn("ws dropped malformed frame");
    }
  }

  private scheduleReconnect() {
    if (this.closed) return;
    const delay = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** this.attempt) + Math.random() * JITTER_MS;
    this.attempt += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      this.connect();
    }, delay);
  }
}
