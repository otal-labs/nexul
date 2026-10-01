import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ScreenShareTile } from "@/components/chat/ScreenShareTile";

vi.mock("@livekit/components-react", () => ({ FocusLayout: () => <video data-testid="share" /> }));

const trackRef = {} as Parameters<typeof ScreenShareTile>[0]["trackRef"];

// jsdom ships no Fullscreen API, so the browser's contract is stood in by a small fake that
// tracks `document.fullscreenElement` and announces changes the way a real browser does.
let isFullscreen: boolean;
const requestFullscreen = vi.fn(function (this: Element) {
  isFullscreen = true;
  document.dispatchEvent(new Event("fullscreenchange"));
  return Promise.resolve();
});
const exitFullscreen = vi.fn(() => {
  isFullscreen = false;
  document.dispatchEvent(new Event("fullscreenchange"));
  return Promise.resolve();
});

const stubFullscreen = (enabled: boolean) => {
  Object.defineProperty(document, "fullscreenEnabled", { value: enabled, configurable: true });
  Object.defineProperty(document, "fullscreenElement", { get: () => (isFullscreen ? requestFullscreen.mock.contexts.at(-1) : null), configurable: true });
  document.exitFullscreen = exitFullscreen;
  HTMLElement.prototype.requestFullscreen = requestFullscreen;
};

beforeEach(() => {
  isFullscreen = false;
  requestFullscreen.mockClear();
  exitFullscreen.mockClear();
  stubFullscreen(true);
});

afterEach(() => {
  // @ts-expect-error restoring jsdom's absence of the API
  delete document.fullscreenEnabled;
  // @ts-expect-error restoring jsdom's absence of the API
  delete document.fullscreenElement;
});

describe("ScreenShareTile", () => {
  it("full-screens the tile's own element, so only the shared screen fills the monitor", async () => {
    const user = userEvent.setup();
    render(<ScreenShareTile trackRef={trackRef} />);

    await user.click(screen.getByRole("button", { name: "Full screen" }));

    expect(requestFullscreen).toHaveBeenCalledTimes(1);
    const target = requestFullscreen.mock.contexts[0] as HTMLElement;
    expect(target).toContainElement(screen.getByTestId("share"));
    expect(target).not.toBe(document.body);
  });

  it("offers Exit full screen once the browser is full screen, and that exits", async () => {
    const user = userEvent.setup();
    render(<ScreenShareTile trackRef={trackRef} />);

    await user.click(screen.getByRole("button", { name: "Full screen" }));
    await user.click(screen.getByRole("button", { name: "Exit full screen" }));

    expect(exitFullscreen).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Full screen" })).toBeInTheDocument();
  });

  it("follows the browser leaving full screen on its own, such as Esc", async () => {
    const user = userEvent.setup();
    render(<ScreenShareTile trackRef={trackRef} />);
    await user.click(screen.getByRole("button", { name: "Full screen" }));

    act(() => {
      isFullscreen = false;
      document.dispatchEvent(new Event("fullscreenchange"));
    });

    expect(screen.getByRole("button", { name: "Full screen" })).toBeInTheDocument();
  });

  it("toggles on double click too", async () => {
    const user = userEvent.setup();
    render(<ScreenShareTile trackRef={trackRef} />);

    await user.dblClick(screen.getByTestId("share"));

    expect(requestFullscreen).toHaveBeenCalledTimes(1);
  });

  it("shows no button where the browser cannot full screen, such as iPhone Safari", () => {
    stubFullscreen(false);
    render(<ScreenShareTile trackRef={trackRef} />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
