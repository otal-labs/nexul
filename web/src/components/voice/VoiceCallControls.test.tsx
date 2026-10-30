import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { VoiceCallControls } from "@/components/voice/VoiceCallControls";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useVoiceStageViewStore } from "@/stores/voiceStageViewStore";

const initialState = useVoiceCallStore.getState();

beforeEach(() => {
  useVoiceCallStore.setState(initialState, true);
  useVoiceStageViewStore.setState({ view: "grid" });
});

describe("VoiceCallControls", () => {
  it("marks a muted microphone as the pressed control and unmutes it", async () => {
    const user = userEvent.setup();
    const toggleMic = vi.fn();
    useVoiceCallStore.setState({ micEnabled: false, toggleMic });
    render(<VoiceCallControls />);

    const mic = screen.getByRole("button", { name: "Unmute microphone" });
    expect(mic).toHaveAttribute("aria-pressed", "true");
    await user.click(mic);
    expect(toggleMic).toHaveBeenCalled();
  });

  it("names the share button for stopping while a screen is shared", () => {
    useVoiceCallStore.setState({ screenShareEnabled: true });
    render(<VoiceCallControls />);

    expect(screen.getByRole("button", { name: "Stop sharing" })).toHaveAttribute("aria-pressed", "true");
  });

  it("leaves the call", async () => {
    const user = userEvent.setup();
    const leave = vi.fn();
    useVoiceCallStore.setState({ leave });
    render(<VoiceCallControls />);

    await user.click(screen.getByRole("button", { name: "Leave call" }));
    expect(leave).toHaveBeenCalled();
  });

  it("switches between the grid and the speaker view, naming the view it switches to", async () => {
    const user = userEvent.setup();
    render(<VoiceCallControls />);

    await user.click(screen.getByRole("button", { name: "Speaker view" }));
    expect(useVoiceStageViewStore.getState().view).toBe("speaker");
    expect(screen.getByRole("button", { name: "Grid view" })).toHaveAttribute("aria-pressed", "true");
  });
});
