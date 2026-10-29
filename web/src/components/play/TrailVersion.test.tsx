import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TrailFacts } from "@/components/play/TrailFacts";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import type { Trail } from "@/models/Trail";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: "u-1",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "done",
  started_at: "2026-10-29T10:00:00Z",
  ended_at: "2026-10-29T10:05:00Z",
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

const keptVersions = (...numbers: number[]) =>
  numbers.map((version, i) => ({ version, created_at: `2026-10-2${8 - i}T10:00:00Z`, name: "", editor_ids: [], via: "", current: i === 0 }));

const renderFacts = (run: Trail) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["plays:read", "plays:write"] });
  render(
    <QueryClientProvider client={client}>
      <TrailFacts trail={run} />
    </QueryClientProvider>,
  );
};

const answer = (versions: object[]) => async (url: string) => {
  if (url === "/api/plays/play-1/history") return { data: { versions, changes: [] } };
  if (url.startsWith("/api/plays/play-1/versions/")) return { data: { version: Number(url.split("/").at(-1)), body: "Read the ticket first." } };
  if (url === "/api/permissions/explain") return { data: { effective: [] } };
  if (url === "/api/workspaces/ws-1/people") return { data: { people: [] } };
  return { data: [] };
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
});

describe("a trail's instructions version", () => {
  it("opens the play's history on that version while it is still kept", async () => {
    vi.mocked(api.get).mockImplementation(answer(keptVersions(8, 7)));
    const user = userEvent.setup();
    renderFacts(trail({ instructions_version: 7 }));

    await user.click(await screen.findByRole("button", { name: "Ran with version 7" }));

    expect(await screen.findByRole("heading", { name: "History" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { current: true })).not.toHaveAccessibleName(/current version/);
    expect(vi.mocked(api.get)).toHaveBeenCalledWith("/api/plays/play-1/versions/7");
  });

  it("says the version is no longer kept, and opens nothing, once a prune has dropped it", async () => {
    vi.mocked(api.get).mockImplementation(answer(keptVersions(18, 17)));
    renderFacts(trail({ instructions_version: 7 }));

    expect(await screen.findByText("Version 7 is no longer kept")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Ran with/ })).not.toBeInTheDocument();
  });

  it("shows nothing for a run from before history", () => {
    vi.mocked(api.get).mockImplementation(answer(keptVersions(8, 7)));
    renderFacts(trail({}));

    expect(screen.queryByText(/version/i)).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/plays/play-1/history");
  });
});
