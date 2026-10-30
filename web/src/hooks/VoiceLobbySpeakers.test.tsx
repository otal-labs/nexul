import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useVoiceLobbySpeakers } from "@/hooks/VoiceHooks";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() } }));

// livekit-client's Room does real WebRTC signaling, which jsdom has no; the fake keeps its listeners so a test can
// fire the events LiveKit would.
const fake = vi.hoisted(() => {
  type Handler = (...args: unknown[]) => void;
  class FakeRoom {
    static all: FakeRoom[] = [];
    handlers = new Map<string, Handler>();
    remoteParticipants = new Map<string, unknown>();
    connect = vi.fn(async () => undefined);
    disconnect = vi.fn();
    constructor() {
      FakeRoom.all.push(this);
    }
    on(event: string, handler: Handler) {
      this.handlers.set(event, handler);
      return this;
    }
    emit(event: string, ...args: unknown[]) {
      this.handlers.get(event)?.(...args);
    }
  }
  return { FakeRoom };
});

vi.mock("livekit-client", () => ({
  Room: fake.FakeRoom,
  RoomEvent: { ParticipantConnected: "participantConnected", TrackPublished: "trackPublished", ActiveSpeakersChanged: "activeSpeakersChanged" },
  Track: { Kind: { Audio: "audio", Video: "video" } },
  setLogLevel: vi.fn(),
}));

const publication = (kind: "audio" | "video", trackSid: string) => ({ kind, trackSid, setSubscribed: vi.fn(), setEnabled: vi.fn() });

const connected = async (index = 0) => {
  await waitFor(() => expect(fake.FakeRoom.all[index]?.connect).toHaveBeenCalled());
  return fake.FakeRoom.all[index]!;
};

beforeEach(() => {
  fake.FakeRoom.all = [];
  vi.mocked(api.post).mockReset();
  vi.mocked(api.post).mockResolvedValue({ data: { ws_url: "wss://lk.example.com", token: "listen-token" } });
});

describe("useVoiceLobbySpeakers", () => {
  it("connects hidden with the listen token and rings whoever LiveKit reports speaking", async () => {
    const { result } = renderHook(() => useVoiceLobbySpeakers("c1", true));
    const room = await connected();

    expect(api.post).toHaveBeenCalledWith("/api/voice/c1/listen");
    expect(room.connect).toHaveBeenCalledWith("wss://lk.example.com", "listen-token", { autoSubscribe: false });
    expect([...result.current]).toEqual([]);

    act(() => room.emit("activeSpeakersChanged", [{ identity: "u2" }, { identity: "u3" }]));
    expect([...result.current]).toEqual(["u2", "u3"]);

    act(() => room.emit("activeSpeakersChanged", []));
    expect([...result.current]).toEqual([]);
  });

  it("subscribes to audio only and switches it off, so no media is forwarded", async () => {
    renderHook(() => useVoiceLobbySpeakers("c1", true));
    const room = await connected();
    const mic = publication("audio", "t-mic");
    const camera = publication("video", "t-cam");

    act(() => room.emit("participantConnected", { trackPublications: new Map([["t-mic", mic], ["t-cam", camera]]) }));
    act(() => room.emit("trackPublished", mic));

    expect(mic.setSubscribed).toHaveBeenCalledOnce();
    expect(mic.setSubscribed).toHaveBeenCalledWith(true);
    expect(mic.setEnabled).toHaveBeenCalledWith(false);
    expect(camera.setSubscribed).not.toHaveBeenCalled();
  });

  it("disconnects on unmount", async () => {
    const { unmount } = renderHook(() => useVoiceLobbySpeakers("c1", true));
    const room = await connected();

    unmount();
    expect(room.disconnect).toHaveBeenCalled();
  });

  it("disconnects and rings nobody once the room empties or you join", async () => {
    const { result, rerender } = renderHook(({ listening }) => useVoiceLobbySpeakers("c1", listening), { initialProps: { listening: true } });
    const room = await connected();
    act(() => room.emit("activeSpeakersChanged", [{ identity: "u2" }]));

    rerender({ listening: false });
    expect(room.disconnect).toHaveBeenCalled();
    expect([...result.current]).toEqual([]);
  });

  it("keeps one listener: moving to another channel disconnects the first before the next connects", async () => {
    const { rerender } = renderHook(({ id }) => useVoiceLobbySpeakers(id, true), { initialProps: { id: "c1" } });
    const first = await connected();

    rerender({ id: "c2" });
    const second = await connected(1);
    expect(first.disconnect).toHaveBeenCalled();
    expect(api.post).toHaveBeenLastCalledWith("/api/voice/c2/listen");
    expect(second.disconnect).not.toHaveBeenCalled();
  });

  it("shows no rings and no error when the listen fails", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("no connector"));
    const { result } = renderHook(() => useVoiceLobbySpeakers("c1", true));

    await waitFor(() => expect(api.post).toHaveBeenCalled());
    expect([...result.current]).toEqual([]);
    expect(fake.FakeRoom.all).toHaveLength(0);
  });

  it("does not connect when a stop lands before the token arrives", async () => {
    let release: (value: unknown) => void = () => undefined;
    vi.mocked(api.post).mockReturnValue(new Promise((resolve) => (release = resolve)) as never);
    const { unmount } = renderHook(() => useVoiceLobbySpeakers("c1", true));

    unmount();
    release({ data: { ws_url: "wss://lk.example.com", token: "late" } });
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(fake.FakeRoom.all).toHaveLength(0);
  });
});
