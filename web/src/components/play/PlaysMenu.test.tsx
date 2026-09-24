import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlaysMenu } from "@/components/play/PlaysMenu";
import type { Play } from "@/models/Play";
import type { Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const docPlay: Play = {
  id: "play-doc",
  workspace_id: "ws-1",
  label: "To tickets via AI",
  type: "doc",
  description: "Turns this doc into backlog tickets",
  instructions: "",
  enabled: true,
  show_when_stage: null,
  excluded_project_ids: [],
  created_by: "u-1",
  created_at: "",
  updated_at: "",
};

const runningTrail = (starterId: string): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-doc",
  play_label: "To tickets via AI",
  target_type: "doc",
  target_id: "doc-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: starterId,
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  move_to_status_id: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "running",
  started_at: "2026-09-17T10:00:00Z",
  ended_at: null,
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
});

interface Scenario {
  permissions?: string[];
  plays?: Play[];
  resolve?: unknown;
  presence?: Record<string, string>;
  trails?: Trail[];
}

const mockApi = ({
  permissions = ["plays:run", "docs:thread"],
  plays = [docPlay],
  resolve = { ok: true, computer_id: "c-1" },
  presence = { "c-1": "connected" },
  trails = [],
}: Scenario) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/auth/me") return { data: { user: { id: "u-me", login: "me" } } };
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: plays };
    if (url === "/api/pairing/resolve") return { data: resolve };
    if (url === "/api/pairing/presence") return { data: { computers: presence } };
    if (url === "/api/pairing/computers") return { data: { computers: [{ id: "c-1", name: "Onik's PC" }] } };
    if (url === "/api/pairing/computers/c-1/providers") return { data: { providers: [] } };
    if (url === "/api/plays/runs") return { data: trails };
    if (url === "/api/plays/latest-choices")
      return { data: { memory_ids: [], move_to_status_id: "", computer_id: "", provider: "", model: "" } };
    if (url === "/api/memories") return { data: [] };
    if (url === "/api/statuses") return { data: [] };
    return { data: [] };
  });

const renderMenu = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PlaysMenu workspaceId="ws-1" projectId="p-1" docId="doc-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlaysMenu", () => {
  it("is hidden without plays:run", async () => {
    mockApi({ permissions: ["docs:thread"] });
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByRole("button", { name: /Plays/ })).not.toBeInTheDocument();
  });

  it("is hidden without docs:thread", async () => {
    mockApi({ permissions: ["plays:run"] });
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByRole("button", { name: /Plays/ })).not.toBeInTheDocument();
  });

  it("is hidden when no doc play applies", async () => {
    mockApi({ plays: [] });
    renderMenu();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything()));
    expect(screen.queryByRole("button", { name: /Plays/ })).not.toBeInTheDocument();
  });

  it("lists the doc plays only, fetched with type doc", async () => {
    const user = userEvent.setup();
    mockApi({});
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", {
      params: { project_id: "p-1", type: "doc" },
    });
    expect(await screen.findByRole("button", { name: /To tickets via AI/ })).toBeInTheDocument();
    expect(screen.getByText("Turns this doc into backlog tickets")).toBeInTheDocument();
  });

  it("shows the readiness reason once and disables the rows", async () => {
    const user = userEvent.setup();
    mockApi({ resolve: { ok: false, reason: "unpaired" } });
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    expect(await screen.findAllByText("Pair a harness in Settings to run plays")).toHaveLength(1);
    expect(screen.getByRole("button", { name: /To tickets via AI/ })).toBeDisabled();
  });

  it("shows 'a run is in progress' once while a trail is active, with no Stop row for another starter", async () => {
    const user = userEvent.setup();
    mockApi({ trails: [runningTrail("u-other")] });
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    expect(await screen.findAllByText("a run is in progress")).toHaveLength(1);
    expect(screen.getByRole("button", { name: /To tickets via AI/ })).toBeDisabled();
    expect(screen.queryByRole("button", { name: /Stop/ })).not.toBeInTheDocument();
  });

  it("gives the starter a Stop row for the active trail", async () => {
    const user = userEvent.setup();
    mockApi({ trails: [runningTrail("u-me")] });
    vi.mocked(api.post).mockResolvedValue({ data: { ...runningTrail("u-me"), state: "interrupted" } });
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    await user.click(await screen.findByRole("button", { name: /Stop To tickets via AI/ }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/stop");
  });

  it("gives a plays:write holder a Stop row even without starting the run", async () => {
    const user = userEvent.setup();
    mockApi({ permissions: ["plays:run", "docs:thread", "plays:write"], trails: [runningTrail("u-other")] });
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    expect(await screen.findByRole("button", { name: /Stop To tickets via AI/ })).toBeInTheDocument();
  });

  it("choosing a play opens the run dialog for the doc target, with no move-to section", async () => {
    const user = userEvent.setup();
    mockApi({});
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    await user.click(await screen.findByRole("button", { name: /To tickets via AI/ }));

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "To tickets via AI" })).toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "On success, move to" })).not.toBeInTheDocument();
    expect(screen.queryByText(/Never moves backwards/)).not.toBeInTheDocument();
  });

  it("flips to 'a run is in progress' from a live play.run frame keyed on the doc target", async () => {
    const user = userEvent.setup();
    mockApi({ trails: [{ ...runningTrail("u-other"), state: "done", ended_at: "2026-09-17T10:05:00Z" }] });
    renderMenu();

    await user.click(await screen.findByRole("button", { name: /Plays/ }));
    expect(await screen.findByRole("button", { name: /To tickets via AI/ })).toBeEnabled();

    usePlayRunStore.getState().applyFrame({
      trail_id: "tr-1", play_id: "play-doc", target_type: "doc", target_id: "doc-1", state: "running", activity: null, ended_at: null, last_error: "",
    });
    expect(await screen.findByText("a run is in progress")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /To tickets via AI/ })).toBeDisabled();
  });
});
