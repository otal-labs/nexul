import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { PhoneBanner } from "@/components/PhoneBanner";
import { usePhoneBannerStore } from "@/stores/phoneBannerStore";

const desktopAgent = "Mozilla/5.0 (X11; Linux x86_64) Chrome/130.0";
const androidAgent = "Mozilla/5.0 (Linux; Android 15; Pixel 9) Chrome/130.0 Mobile";
const iphoneAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Safari/605.1";

const stubViewport = (phone: boolean) =>
  vi.stubGlobal(
    "matchMedia",
    vi.fn().mockReturnValue({ matches: phone, addEventListener: vi.fn(), removeEventListener: vi.fn() }),
  );

const stubUserAgent = (value: string) => Object.defineProperty(navigator, "userAgent", { value, configurable: true });

beforeEach(() => {
  localStorage.clear();
  usePhoneBannerStore.setState({ dismissed: false });
  stubUserAgent(desktopAgent);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("PhoneBanner", () => {
  it("shows below 768px", () => {
    stubViewport(true);
    render(<PhoneBanner />);
    expect(screen.getByRole("status")).toHaveTextContent("Nexul is built for tablet and desktop.");
  });

  it("stays away at 768px and up", () => {
    stubViewport(false);
    render(<PhoneBanner />);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("links Android phones to the releases page", () => {
    stubViewport(true);
    stubUserAgent(androidAgent);
    render(<PhoneBanner />);
    expect(screen.getByRole("link", { name: "Get the Android app" })).toHaveAttribute("href", "https://github.com/otal-labs/nexul/releases");
  });

  it("gives other phones no link", () => {
    stubViewport(true);
    stubUserAgent(iphoneAgent);
    render(<PhoneBanner />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("dismisses and remembers it on this device", async () => {
    const user = userEvent.setup();
    stubViewport(true);
    render(<PhoneBanner />);
    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByRole("status")).not.toBeInTheDocument();

    const stored = JSON.parse(localStorage.getItem("phone-banner") ?? "{}") as { state?: { dismissed?: boolean } };
    expect(stored.state?.dismissed).toBe(true);
  });
});
