import { RoomContext } from "@livekit/components-react";
import type { Room } from "livekit-client";

import { VoiceCallControls } from "@/components/voice/VoiceCallControls";
import { VoiceStage } from "@/components/voice/VoiceStage";

interface ConnectedCallProps {
  room: Room;
}

// Stacked over the chat when narrow; beside it, at the panel's full height, from 1000px (the thread's container query).
export const ConnectedCall = ({ room }: ConnectedCallProps) => {
  return (
    <RoomContext.Provider value={room}>
      <div className="h-[clamp(260px,46vh,460px)] shrink-0 border-b border-border p-3 has-[[data-screen-share-tile]]:h-[clamp(320px,66vh,640px)] @[1000px]/thread:h-auto @[1000px]/thread:flex-1 @[1000px]/thread:border-b-0">
        <div role="region" aria-label="Call" className="flex size-full flex-col gap-3 rounded-lg bg-surface-2 p-3">
          <VoiceStage />
          <VoiceCallControls />
        </div>
      </div>
    </RoomContext.Provider>
  );
};
