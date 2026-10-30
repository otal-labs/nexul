import { render, screen } from "@testing-library/react";
import type { Participant } from "livekit-client";
import { ConnectionQuality } from "livekit-client";
import { describe, expect, it, vi } from "vitest";

import { VoicePersonRow } from "@/components/voice/VoicePersonRow";

const livekit = vi.hoisted(() => ({ quality: "good" }));
vi.mock("@livekit/components-react", () => ({
  useIsSpeaking: () => false,
  useIsMuted: () => false,
  useConnectionQualityIndicator: () => ({ quality: livekit.quality }),
}));
vi.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: () => (identity: string) => ({ user_id: identity, login: identity, display_name: "", avatar_url: "" }),
}));

const participant = { identity: "u2", name: "Lena Kowalski", isLocal: false } as unknown as Participant;

describe("VoicePersonRow connection mark", () => {
  it.each([
    [ConnectionQuality.Poor, "Weak connection"],
    [ConnectionQuality.Lost, "Connection lost"],
  ])("labels a %s connection", (quality, label) => {
    livekit.quality = quality;
    render(<VoicePersonRow participant={participant} />);

    expect(screen.getByLabelText(label)).toBeInTheDocument();
  });

  it("shows nothing while the connection is good", () => {
    livekit.quality = ConnectionQuality.Good;
    render(<VoicePersonRow participant={participant} />);

    expect(screen.queryByLabelText(/connection/i)).not.toBeInTheDocument();
  });
});
