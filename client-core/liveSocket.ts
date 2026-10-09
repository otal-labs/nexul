// The reconnecting live-events socket both apps hold; each app decides when it is open and how it catches up.
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
  send?: (data: string) => void;
  close: () => void;
  readyState?: number;
}

export type LiveEventsHandler = (frame: ServerFrame) => void;

export interface LiveEventsClientOptions {
  wsFactory?: (url: string) => LiveSocket;
  // Fired on every open after the first, never the initial connect: frames sent while it was down are lost, so catch up.
  onReconnect?: () => void;
}

const MAX_BACKOFF_MS = 30_000;
const BASE_BACKOFF_MS = 500;
const JITTER_MS = 250;

export const parseFrame = (raw: string): ServerFrame => {
  const data = JSON.parse(raw) as unknown;
  if (typeof data !== "object" || data === null) throw new Error("frame is not an object");
  const { topic, type, payload } = data as Record<string, unknown>;
  if (typeof topic !== "string" || typeof type !== "string") throw new Error("frame missing topic/type");
  return { topic, type, payload };
};

// WebSocket.CONNECTING, spelled out so the rule reads no global.
const CONNECTING = 0;

export const closeSocket = (socket: LiveSocket) => {
  if (socket.readyState === CONNECTING) {
    // Closing mid-handshake logs a Chrome warning, so let it finish opening, then close.
    socket.onopen = () => socket.close();
    return;
  }
  socket.onopen = null;
  socket.close();
};

export class LiveEventsClient {
  private socket: LiveSocket | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private attempt = 0;
  private closed = false;
  private everOpened = false;
  private readonly handlers = new Set<LiveEventsHandler>();
  private readonly wsFactory: (url: string) => LiveSocket;
  private readonly onReconnect: (() => void) | undefined;

  constructor(
    private readonly url: string,
    opts: LiveEventsClientOptions = {},
  ) {
    this.wsFactory = opts.wsFactory ?? ((u) => new WebSocket(u) as unknown as LiveSocket);
    this.onReconnect = opts.onReconnect;
  }

  connect() {
    if (this.closed) return;
    const socket = this.wsFactory(this.url);
    this.socket = socket;
    socket.onopen = () => {
      this.attempt = 0;
      if (this.everOpened) this.onReconnect?.();
      this.everOpened = true;
    };
    socket.onmessage = (ev) => this.handleMessage(ev.data);
    // The URL carries the session token, so it stays out of the log; a drop is retried, so it warns.
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
      // Detach handlers before closing, or a torn-down socket's error/close events fire and trigger a reconnect.
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      closeSocket(socket);
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
