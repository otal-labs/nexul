import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ServerUpdatedBanner } from "@/components/ServerUpdatedBanner";
import { useServerUpdateStore } from "@/stores/serverUpdateStore";

const reload = vi.fn();

beforeEach(() => {
  useServerUpdateStore.setState({ pendingVersion: null });
  vi.stubGlobal("location", { ...window.location, reload });
});

afterEach(() => {
  vi.unstubAllGlobals();
  reload.mockReset();
});

describe("ServerUpdatedBanner", () => {
  it("stays hidden until an update is pending", () => {
    render(<ServerUpdatedBanner />);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("names the new version and reloads the page on Reload", async () => {
    const user = userEvent.setup();
    useServerUpdateStore.setState({ pendingVersion: "v0.2.0-beta-331" });
    render(<ServerUpdatedBanner />);
    expect(screen.getByRole("status")).toHaveTextContent("Nexul updated to v0.2.0-beta-331. Reload to use it.");

    await user.click(screen.getByRole("button", { name: "Reload" }));
    expect(reload).toHaveBeenCalledOnce();
  });

  it("hides on Dismiss", async () => {
    const user = userEvent.setup();
    useServerUpdateStore.setState({ pendingVersion: "v0.2.0-beta-331" });
    render(<ServerUpdatedBanner />);
    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
