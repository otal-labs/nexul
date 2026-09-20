import { create } from "zustand";
import { toast } from "sonner";
import type { Room } from "livekit-client";

import { api, errorMessage } from "@/api/client";
import { isVoiceNotConfiguredError, type VoiceJoinToken } from "@/models/Voice";

export type VoiceCallStatus = "idle" | "connecting" | "connected" | "not_configured" | "error";

// Connection lifecycle is imperative async work driven from LiveKit's callbacks, so it lives in store actions (F5).
export type VoiceCallStore = {
  activeConversationId: string | null;
  status: VoiceCallStatus;
  room: Room | null;
  error: string | undefined;
  micEnabled: boolean;
  cameraEnabled: boolean;
  screenShareEnabled: boolean;
  join: (conversationId: string) => Promise<void>;
  leave: () => void;
  retry: () => void;
  toggleMic: () => void;
  toggleCamera: () => void;
  toggleScreenShare: () => void;
};

// Monotonic guard: any teardown or newer join bumps it; every await in join() re-checks before touching state.
let joinSeq = 0;

// Best-effort teardown: tell the server immediately instead of ghosting until the next presence poll.
const disconnectRoom = (room: Room | null, conversationId: string | null) => {
  room?.disconnect();
  if (conversationId) void api.post(`/api/voice/${conversationId}/leave`).catch(() => undefined);
};

export const useVoiceCallStore = create<VoiceCallStore>((set, get) => ({
  activeConversationId: null,
  status: "idle",
  room: null,
  error: undefined,
  micEnabled: false,
  cameraEnabled: false,
  screenShareEnabled: false,

  join: async (conversationId) => {
    const prev = get();
    const alreadyOn = prev.activeConversationId === conversationId;
    if (alreadyOn && (prev.status === "connected" || prev.status === "connecting")) return;
    const seq = ++joinSeq;
    disconnectRoom(prev.room, prev.room ? prev.activeConversationId : null);
    set({ activeConversationId: conversationId, status: "connecting", room: null, error: undefined });
    try {
      const [{ data }, livekit] = await Promise.all([
        api.post<VoiceJoinToken>(`/api/voice/${conversationId}/token`),
        import("livekit-client"),
      ]);
      // livekit-client logs connection/track chatter at info by default — noise in the app console.
      livekit.setLogLevel("warn");
      if (seq !== joinSeq) return;
      const room = new livekit.Room();
      room.on(livekit.RoomEvent.Disconnected, (reason) => {
        if (seq !== joinSeq) return; // an intentional leave/rejoin already handled state
        set({ activeConversationId: null, status: "idle", room: null });
        if (reason !== livekit.DisconnectReason.CLIENT_INITIATED) {
          toast.error("Disconnected from the voice call");
        }
      });
      await room.connect(data.ws_url, data.token);
      if (seq !== joinSeq) {
        room.disconnect();
        return;
      }
      await room.localParticipant.setMicrophoneEnabled(true).catch(() => undefined);
      set({
        room,
        status: "connected",
        micEnabled: room.localParticipant.isMicrophoneEnabled,
        cameraEnabled: room.localParticipant.isCameraEnabled,
        screenShareEnabled: room.localParticipant.isScreenShareEnabled,
      });
    } catch (err) {
      if (seq !== joinSeq) return;
      set({ status: isVoiceNotConfiguredError(err) ? "not_configured" : "error", error: errorMessage(err) });
    }
  },

  leave: () => {
    joinSeq++;
    const { room, activeConversationId } = get();
    disconnectRoom(room, activeConversationId);
    set({
      activeConversationId: null,
      status: "idle",
      room: null,
      error: undefined,
      micEnabled: false,
      cameraEnabled: false,
      screenShareEnabled: false,
    });
  },

  retry: () => {
    const id = get().activeConversationId;
    if (id) void get().join(id);
  },

  toggleMic: () => {
    const { room } = get();
    if (!room) return;
    const next = !room.localParticipant.isMicrophoneEnabled;
    void room.localParticipant.setMicrophoneEnabled(next);
    set({ micEnabled: next });
  },

  toggleCamera: () => {
    const { room } = get();
    if (!room) return;
    const next = !room.localParticipant.isCameraEnabled;
    void room.localParticipant.setCameraEnabled(next);
    set({ cameraEnabled: next });
  },

  toggleScreenShare: () => {
    const { room } = get();
    if (!room) return;
    const next = !room.localParticipant.isScreenShareEnabled;
    void room.localParticipant.setScreenShareEnabled(next);
    set({ screenShareEnabled: next });
  },
}));
