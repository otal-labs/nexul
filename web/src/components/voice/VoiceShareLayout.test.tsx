import { render, screen } from "@testing-library/react";
import type { TrackReference } from "@livekit/components-core";
import { describe, expect, it, vi } from "vitest";

import { VoiceShareLayout } from "@/components/voice/VoiceShareLayout";

vi.mock("@livekit/components-react", () => ({ useParticipants: () => [] }));
vi.mock("@/components/chat/ScreenShareTile", () => ({ ScreenShareTile: () => <video data-testid="share" /> }));

const shareBy = (isLocal: boolean) => ({ participant: { name: "Bob Tanaka", identity: "u2", isLocal } }) as unknown as TrackReference;

describe("VoiceShareLayout", () => {
  it("plays someone else's screen and names whose it is under it", () => {
    render(<VoiceShareLayout share={shareBy(false)} />);

    expect(screen.getByTestId("share")).toBeInTheDocument();
    expect(screen.getByText("Bob Tanaka's screen")).toBeInTheDocument();
  });

  it("doesn't play your own screen back to you, and offers Stop sharing instead", () => {
    render(<VoiceShareLayout share={shareBy(true)} />);

    expect(screen.queryByTestId("share")).not.toBeInTheDocument();
    expect(screen.getByText("You're sharing your screen")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop sharing" })).toBeInTheDocument();
  });
});
