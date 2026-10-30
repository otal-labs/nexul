import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { useSessionStore } from "@/stores/sessionStore";

const surface = () => screen.getByText("content").parentElement;

describe("ShowcaseSurface", () => {
  afterEach(() => useSessionStore.setState({ token: null, isLoggedIn: false }));

  it("takes the app frame's place signed in, and fills the screen when asked to", () => {
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    const { unmount } = render(<ShowcaseSurface quiet>content</ShowcaseSurface>);
    expect(surface()).toHaveClass("h-full", "rounded-xl");
    unmount();

    render(<ShowcaseSurface quiet fullScreen>content</ShowcaseSurface>);
    expect(surface()).toHaveClass("h-dvh");
    expect(surface()).not.toHaveClass("rounded-xl");
  });
});
