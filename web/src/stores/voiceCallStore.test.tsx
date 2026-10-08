import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

vi.mock("@/api/client", () => ({
  api: { post: vi.fn() },
  errorMessage: (error: unknown) => (error as Error)?.message ?? "Something went wrong",
}));

// livekit-client's Room does real WebRTC signaling — not available in jsdom (so:
// mock the connection layer, never exercise real WebRTC). The store loads it via dynamic import,
// which vi.mock intercepts the same as a static one.
const connect = vi.fn(async () => undefined);
const disconnect = vi.fn();
const on = vi.fn();
const setMicrophoneEnabled = vi.fn(async () => undefined);
const remoteParticipants = new Map<string, object>();

vi.mock("livekit-client", () => ({
  Room: vi.fn().mockImplementation(function () {
    return {
      connect,
      disconnect,
      on,
      remoteParticipants,
      localParticipant: {
        isMicrophoneEnabled: true,
        isCameraEnabled: false,
        isScreenShareEnabled: false,
        setMicrophoneEnabled,
        setCameraEnabled: vi.fn(),
        setScreenShareEnabled: vi.fn(),
      },
    };
  }),
  RoomEvent: { Disconnected: "disconnected", ParticipantConnected: "participantConnected", ParticipantDisconnected: "participantDisconnected" },
  DisconnectReason: { CLIENT_INITIATED: 0 },
  setLogLevel: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { error: vi.fn() }) }));

const initialState = useVoiceCallStore.getState();

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  connect.mockClear();
  disconnect.mockClear();
  on.mockClear();
  remoteParticipants.clear();
  useVoiceCallStore.setState(initialState, true);
});

describe("voiceCallStore", () => {
  it("join mints a token, connects, enables the mic, and lands connected", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk", token: "tok" } });

    await useVoiceCallStore.getState().join("c1", "huddle");

    const s = useVoiceCallStore.getState();
    expect(api.post).toHaveBeenCalledWith("/api/voice/c1/token");
    expect(connect).toHaveBeenCalledWith("wss://lk", "tok");
    expect(setMicrophoneEnabled).toHaveBeenCalledWith(true);
    expect(s.status).toBe("connected");
    expect(s.activeConversationId).toBe("c1");
    expect(s.micEnabled).toBe(true);
    expect(s.room).not.toBeNull();
  });

  it("join surfaces the owner-setup-needed state on the LiveKit-not-configured error", async () => {
    vi.mocked(api.post).mockRejectedValue({
      message: "Request failed",
      response: { data: { code: "INVALID", message: "voice channels need a LiveKit connector configured by the workspace owner" } },
    });

    await useVoiceCallStore.getState().join("c1", "huddle");

    expect(useVoiceCallStore.getState().status).toBe("not_configured");
  });

  it("join failure lands in a retryable error state with the message", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("network unreachable"));

    await useVoiceCallStore.getState().join("c1", "huddle");

    const s = useVoiceCallStore.getState();
    expect(s.status).toBe("error");
    expect(s.error).toBe("network unreachable");
  });

  it("leave disconnects the room, posts the server-side leave, and resets state", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk", token: "tok" } });
    await useVoiceCallStore.getState().join("c1", "huddle");

    useVoiceCallStore.getState().leave();

    const s = useVoiceCallStore.getState();
    expect(disconnect).toHaveBeenCalled();
    expect(api.post).toHaveBeenCalledWith("/api/voice/c1/leave");
    expect(s.status).toBe("idle");
    expect(s.activeConversationId).toBeNull();
    expect(s.room).toBeNull();
  });

  it("an unexpected disconnect resets state and toasts", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk", token: "tok" } });
    await useVoiceCallStore.getState().join("c1", "huddle");
    const { toast } = await import("sonner");
    const onDisconnected = on.mock.calls.find(([event]) => event === "disconnected")?.[1] as (reason?: number) => void;

    onDisconnected(7);

    expect(useVoiceCallStore.getState().status).toBe("idle");
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("Disconnected from the voice call");
  });

  it("joining the channel already connected is a no-op", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk", token: "tok" } });
    await useVoiceCallStore.getState().join("c1", "huddle");
    vi.mocked(api.post).mockClear();

    await useVoiceCallStore.getState().join("c1", "huddle");

    expect(api.post).not.toHaveBeenCalled();
  });
});

describe("voiceCallStore leaving a call nobody else is in", () => {
  const FIVE_MINUTES = 5 * 60_000;
  const fire = (event: string) => (on.mock.calls.find(([name]) => name === event)?.[1] as () => void)();

  beforeEach(async () => {
    vi.useFakeTimers();
    vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk", token: "tok" } });
    const { toast } = await import("sonner");
    vi.mocked(toast).mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("leaves after five minutes alone, through the normal leave, with the reason recorded and toasted", async () => {
    const { toast } = await import("sonner");
    await useVoiceCallStore.getState().join("c1", "developers");

    vi.advanceTimersByTime(FIVE_MINUTES - 1);
    expect(useVoiceCallStore.getState().status).toBe("connected");
    vi.advanceTimersByTime(1);

    const s = useVoiceCallStore.getState();
    const reason = "You left developers after 5 minutes alone in the call.";
    expect(s.status).toBe("idle");
    expect(disconnect).toHaveBeenCalled();
    expect(api.post).toHaveBeenCalledWith("/api/voice/c1/leave");
    expect(s.leftAlone).toEqual({ conversationId: "c1", reason });
    expect(vi.mocked(toast)).toHaveBeenCalledWith(reason);
  });

  it("someone joining stops the clock, and their leaving starts it over", async () => {
    await useVoiceCallStore.getState().join("c1", "developers");

    vi.advanceTimersByTime(FIVE_MINUTES - 1000);
    remoteParticipants.set("bob", {});
    fire("participantConnected");
    vi.advanceTimersByTime(FIVE_MINUTES);
    expect(useVoiceCallStore.getState().status).toBe("connected");

    remoteParticipants.clear();
    fire("participantDisconnected");
    vi.advanceTimersByTime(FIVE_MINUTES - 1000);
    expect(useVoiceCallStore.getState().status).toBe("connected");
    vi.advanceTimersByTime(1000);
    expect(useVoiceCallStore.getState().status).toBe("idle");
  });

  it("a call that dropped on its own never reports leaving for being alone", async () => {
    const { toast } = await import("sonner");
    await useVoiceCallStore.getState().join("c1", "developers");

    fire("disconnected");
    vi.advanceTimersByTime(FIVE_MINUTES);

    expect(vi.mocked(toast)).not.toHaveBeenCalled();
    expect(useVoiceCallStore.getState().leftAlone).toBeNull();
  });

  it("rejoining clears the reason", async () => {
    await useVoiceCallStore.getState().join("c1", "developers");
    vi.advanceTimersByTime(FIVE_MINUTES);

    await useVoiceCallStore.getState().join("c1", "developers");

    expect(useVoiceCallStore.getState().leftAlone).toBeNull();
  });
});
