import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayRow } from "@/components/play/PlayRow";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Play } from "@/models/Play";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [{ value: "plays:run", label: "Run plays", domain: "plays", action: "run" }];

const play: Play = {
  id: "play-1",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  builtin_key: "",
  created_by: "",
  created_at: "",
  updated_at: "",
};

const mockGet = (grants: unknown[] = []) => (url: string) => {
  if (url === "/api/permissions?resource_type=play&resource_id=play-1") {
    return Promise.resolve({ data: { grants } });
  }
  if (url === "/api/permissions/explain") {
    const source = { kind: "role", entity_type: "workspace", entity_id: "ws-1", target_type: "role", target_id: "r-1", state: "allow", entity_name: "Acme", target_name: "Developer", sentence: "Allowed by role Developer" };
    const verdict = { allowed: true, source };
    return Promise.resolve({ data: { resource_type: "play", resource_id: "play-1", user_id: "", effective: [{ action: "plays:run", ...verdict, fallback: verdict }] } });
  }
  if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people: [{ user_id: "u1", login: "bob", display_name: "", avatar_url: "https://example.com/b.png" }] } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Member", permissions: ["plays:write", "plays:delete"] } });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const renderRow = (props: Partial<{ canWrite: boolean; canDelete: boolean }> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ul>
        <PlayRow
          play={play}
          workspaceId="ws-1"
          canWrite={props.canWrite ?? true}
          canDelete={props.canDelete ?? true}
          onEdit={() => {}}
        />
      </ul>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlayRow", () => {
  it("offers no menu without plays:write or plays:delete", async () => {
    vi.mocked(api.get).mockImplementation(mockGet());
    renderRow({ canWrite: false, canDelete: false });
    await userEvent.setup().pointer({ keys: "[MouseRight]", target: screen.getByText("Fix with AI") });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("opens its items at the pointer on a right-click, Permissions… between Edit and Delete", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(mockGet());
    renderRow();

    await user.pointer({ keys: "[MouseRight]", target: screen.getByText("Fix with AI") });
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
    expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual(["Edit", "Permissions…", "Delete"]);
  });

  it("shows the excluded count from the grants query", async () => {
    vi.mocked(api.get).mockImplementation(
      mockGet([{ resource_type: "play", resource_id: "play-1", user_id: "u1", allow: [], deny: ["plays:run"] }]),
    );
    renderRow();
    expect(await screen.findByText("1 excluded")).toBeInTheDocument();
  });

  it("shows how many rules hide the play", async () => {
    vi.mocked(api.get).mockImplementation(
      mockGet([
        { resource_type: "play", resource_id: "play-1", target: "role", role_id: "r1", user_id: "", allow: [], deny: ["plays:read"] },
        { resource_type: "play", resource_id: "play-1", user_id: "u1", allow: ["plays:read"], deny: [] },
      ]),
    );
    renderRow();
    expect(await screen.findByText("hidden from 1")).toBeInTheDocument();
  });

  it("opens the permissions panel on the play, today's exclusions as Deny on Run", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ resource_type: "play", resource_id: "play-1", user_id: "u1", allow: [], deny: ["plays:run"], created_at: "", updated_at: "" }]),
    );
    renderRow();

    await user.pointer({ keys: "[MouseRight]", target: await screen.findByText("Fix with AI") });
    await user.click(await screen.findByRole("menuitem", { name: "Permissions…" }));
    await user.click(await screen.findByRole("button", { name: /bob/ }));
    const run = await screen.findByRole("radiogroup", { name: "Run" });
    expect(within(run).getByRole("radio", { name: "Deny" })).toBeChecked();
    expect(screen.getByRole("button", { name: "Remove from this play" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });
});
