import { render, screen } from "@testing-library/react";
import type { TrackReferenceOrPlaceholder } from "@livekit/components-core";
import { describe, expect, it, vi } from "vitest";

import { VoiceStageGrid } from "@/components/voice/VoiceStageGrid";

vi.mock("@livekit/components-react", () => ({
  TrackRefContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
  useParticipants: () => [],
  useVisualStableUpdate: (tracks: unknown[]) => tracks,
}));
vi.mock("@/components/voice/VoiceTile", () => ({ VoiceTile: () => <div data-testid="tile" /> }));
// jsdom lays nothing out, so the well is given the size of a short 1024px stage.
vi.mock("@/hooks/useElementSize", () => ({ useElementSize: () => [{ current: null }, { width: 736, height: 280 }] }));

const tracks = Array.from({ length: 9 }, (_, i) => ({ participant: { identity: `u${i}` }, source: "camera" }) as unknown as TrackReferenceOrPlaceholder);

describe("VoiceStageGrid", () => {
  it("draws the people who fit and puts the rest behind a +N tile", () => {
    render(<VoiceStageGrid tracks={tracks} />);

    expect(screen.getAllByTestId("tile")).toHaveLength(7);
    expect(screen.getByRole("button", { name: "2 more in the call" })).toHaveTextContent("+2");
  });
});
