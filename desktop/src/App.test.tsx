import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import type { DesktopBridge } from "../electron/ipc";
import type { PublicVaultState } from "../electron/state";

const makeState = (overrides: Partial<PublicVaultState> = {}): PublicVaultState => ({
  activeId: null,
  instances: [],
  ...overrides,
});

interface BridgeHarness {
  bridge: DesktopBridge;
  push: (state: PublicVaultState) => void;
  importToken: ReturnType<typeof vi.fn>;
  listInstances: ReturnType<typeof vi.fn>;
  connect: ReturnType<typeof vi.fn>;
  remove: ReturnType<typeof vi.fn>;
  refresh: ReturnType<typeof vi.fn>;
  onStateChanged: ReturnType<typeof vi.fn>;
}

const makeBridge = (initial: PublicVaultState = makeState()): BridgeHarness => {
  let listener: ((state: PublicVaultState) => void) | null = null;
  const onStateChanged = vi.fn((cb: (state: PublicVaultState) => void) => {
    listener = cb;
    return () => {
      listener = null;
    };
  });
  const importToken = vi.fn(async (token: string) => {
    if (token.includes("bad")) return { ok: false, error: "not a connection token" };
    listener?.({ ...initial, activeId: "a" });
    return { ok: true, id: "a" };
  });
  const listInstances = vi.fn(async () => initial);
  const connect = vi.fn(async () => ({ ok: true }));
  const remove = vi.fn(async () => ({ ok: true }));
  const refresh = vi.fn(async () => {});
  const push = (state: PublicVaultState) => listener?.(state);
  return { bridge: { onStateChanged, importToken, listInstances, connect, remove, refresh }, push, importToken, listInstances, connect, remove, refresh, onStateChanged };
};

const instance = (overrides: Partial<PublicVaultState["instances"][number]> = {}) => ({
  id: "a",
  instanceUrl: "https://a.example",
  expiresAtSec: 1_800_000_000,
  addedAtSec: 1_700_000_000,
  connection: { status: "idle" as const, error: null },
  ...overrides,
});

describe("App", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows an empty state and subscribes to pushes on mount", async () => {
    const harness = makeBridge();
    render(<App bridge={harness.bridge} />);
    expect(await screen.findByText("No instances yet — import a connection token above.")).toBeInTheDocument();
    expect(harness.listInstances).toHaveBeenCalledTimes(1);
    expect(harness.onStateChanged).toHaveBeenCalledTimes(1);
  });

  it("unsubscribes from pushes on unmount", async () => {
    const harness = makeBridge();
    const { unmount } = render(<App bridge={harness.bridge} />);
    await screen.findByText("No instances yet — import a connection token above.");
    const unsubscribe = harness.onStateChanged.mock.results[0]?.value as () => void;
    const spy = vi.fn(unsubscribe);
    unmount();
    spy();
    expect(spy).toHaveBeenCalled();
  });

  it("renders instances from the initial listing with their real status", async () => {
    const state = makeState({
      activeId: "a",
      instances: [
        instance({ connection: { status: "connected", error: null } }),
        instance({
          id: "b",
          instanceUrl: "https://b.example",
          connection: { status: "unreachable", error: "ECONNREFUSED" },
        }),
      ],
    });
    const harness = makeBridge(state);
    render(<App bridge={harness.bridge} />);
    expect(await screen.findByText("https://a.example")).toBeInTheDocument();
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getByText("https://b.example")).toBeInTheDocument();
    expect(screen.getByText(/Unreachable/)).toBeInTheDocument();
    expect(screen.getByText(/ECONNREFUSED/)).toBeInTheDocument();
  });

  it("updates rendered status when the main process pushes new state", async () => {
    const harness = makeBridge(makeState({ instances: [instance()] }));
    render(<App bridge={harness.bridge} />);
    await screen.findByText("Not checked");
    harness.push(makeState({ activeId: "a", instances: [instance({ connection: { status: "connecting", error: null } })] }));
    expect(await screen.findByText("Connecting…")).toBeInTheDocument();
    harness.push(makeState({ activeId: "a", instances: [instance({ connection: { status: "connected", error: null } })] }));
    expect(await screen.findByText("Connected")).toBeInTheDocument();
  });

  it("imports a token on submit, clearing the input on success", async () => {
    const user = userEvent.setup();
    const harness = makeBridge();
    render(<App bridge={harness.bridge} />);
    await screen.findByText("No instances yet — import a connection token above.");
    const input = screen.getByLabelText("Connection token");
    await user.type(input, "  some.token.value  ");
    await user.click(screen.getByRole("button", { name: "Import" }));
    await waitFor(() => expect(harness.importToken).toHaveBeenCalledWith("some.token.value"));
    expect(input).toHaveValue("");
  });

  it("surfaces the import error and keeps the token", async () => {
    const user = userEvent.setup();
    const harness = makeBridge();
    render(<App bridge={harness.bridge} />);
    await screen.findByText("No instances yet — import a connection token above.");
    await user.type(screen.getByLabelText("Connection token"), "bad.token");
    await user.click(screen.getByRole("button", { name: "Import" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("not a connection token");
    expect(screen.getByLabelText("Connection token")).toHaveValue("bad.token");
  });

  it("disables import for an empty or in-flight token", async () => {
    const user = userEvent.setup();
    const harness = makeBridge();
    render(<App bridge={harness.bridge} />);
    const importButton = screen.getByRole("button", { name: "Import" });
    expect(importButton).toBeDisabled();
    await user.type(screen.getByLabelText("Connection token"), "x");
    expect(importButton).toBeEnabled();
  });

  it("connects and removes via the bridge", async () => {
    const user = userEvent.setup();
    const harness = makeBridge(makeState({ activeId: "a", instances: [instance()] }));
    render(<App bridge={harness.bridge} />);
    await screen.findByText("https://a.example");
    await user.click(screen.getByRole("button", { name: "Connect" }));
    expect(harness.connect).toHaveBeenCalledWith("a");
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(harness.remove).toHaveBeenCalledWith("a");
  });

  it("refreshes connection states on demand", async () => {
    const user = userEvent.setup();
    const harness = makeBridge(makeState({ instances: [instance()] }));
    render(<App bridge={harness.bridge} />);
    await screen.findByText("https://a.example");
    await user.click(screen.getByRole("button", { name: "Refresh" }));
    expect(harness.refresh).toHaveBeenCalledTimes(1);
  });
});
