import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AutomationHistoryDialog } from "@/components/automation/AutomationHistoryDialog";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const bodies: Record<number, string> = {
  3: "a := 1\nb := 3\nreturn a\n",
  2: "a := 1\nb := 2\nreturn a\n",
  1: "a := 1\nreturn a\n",
};

const summary = (version: number, status: "active" | "pending" | "inactive", name: string) => ({
  id: `v${version}`,
  version,
  created_at: `2026-10-0${version}T10:00:00Z`,
  name,
  editor_ids: ["u-1"],
  via: "",
  current: status === "active",
  status,
  words_added: null,
  words_removed: null,
  restored_from: null,
});

const history = {
  versions: [summary(3, "pending", "tighten the retry"), summary(2, "active", ""), summary(1, "inactive", "first cut")],
  changes: [],
};

const renderDialog = (onClose = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["automations:read", "automations:write"] });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <AutomationHistoryDialog automationId="a1" name="Ticket finished" onClose={onClose} />
    </QueryClientProvider>,
  );
  return onClose;
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/automations/a1/history") return { data: history };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["automations:read", "automations:write"] } };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice Moreau", avatar_url: "" }] } };
    const version = Number(url.split("/").at(-1));
    return { data: { version, body: bodies[version] } };
  });
  vi.mocked(api.post).mockResolvedValue({ data: {} });
});

describe("AutomationHistoryDialog", () => {
  it("opens on the pending version, marks its changed line against the one before it, and merges it", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();

    const change = await screen.findByRole("group", { name: /^Change 1 of 1/ });
    expect(change).toHaveTextContent("b :=");
    expect(within(change).getByText("3").tagName).toBe("INS");
    expect(screen.getByText("Pending")).toBeInTheDocument();
    expect(screen.getByText("Current")).toBeInTheDocument();
    expect(screen.getByText("tighten the retry")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Merge #3" }));

    expect(api.post).toHaveBeenCalledWith("/api/automations/a1/versions/v3/merge");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("offers no action on the active version", async () => {
    const user = userEvent.setup();
    renderDialog();
    await screen.findByRole("group", { name: /^Change 1 of 1/ });

    await user.click(screen.getByRole("button", { name: /current version/ }));

    expect(await screen.findByText("This is the active code")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Merge|Rollback/ })).not.toBeInTheDocument();
  });

  it("rolls back to an older version only after confirming", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();
    await screen.findByRole("group", { name: /^Change 1 of 1/ });

    await user.click(screen.getByRole("button", { name: /first cut/ }));
    await user.click(await screen.findByRole("button", { name: "Rollback to #1" }));
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(api.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Rollback to #1" }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    expect(api.post).toHaveBeenCalledWith("/api/automations/a1/versions/v1/rollback");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });
});
