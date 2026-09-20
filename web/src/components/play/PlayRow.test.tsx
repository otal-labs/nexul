import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayRow } from "@/components/play/PlayRow";
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
  created_by: "",
  created_at: "",
  updated_at: "",
};

const mockGet = (grants: unknown[] = []) => (url: string) => {
  if (url === "/api/permissions?resource_type=play&resource_id=play-1") {
    return Promise.resolve({ data: { grants } });
  }
  if (url === "/api/permissions/users") return Promise.resolve({ data: { users: [] } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
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
});

describe("PlayRow", () => {
  it("hides Exclude users without plays:write", () => {
    vi.mocked(api.get).mockImplementation(mockGet());
    renderRow({ canWrite: false, canDelete: false });
    expect(screen.queryByRole("button", { name: /exclude users/i })).not.toBeInTheDocument();
  });

  it("shows the excluded count from the grants query", async () => {
    vi.mocked(api.get).mockImplementation(
      mockGet([{ resource_type: "play", resource_id: "play-1", user_id: "u1", allow: [], deny: ["plays:run"] }]),
    );
    renderRow();
    expect(await screen.findByText("1 excluded")).toBeInTheDocument();
  });

  it("opens the exclusion dialog titled Exclude users, scoped to this play", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(mockGet());
    renderRow();

    await user.click(await screen.findByRole("button", { name: "Exclude users from Fix with AI" }));
    expect(await screen.findByRole("heading", { name: "Exclude users" })).toBeInTheDocument();
    expect(await screen.findByText("Run plays")).toBeInTheDocument();
    await waitFor(() =>
      expect(api.get).toHaveBeenCalledWith("/api/permissions?resource_type=play&resource_id=play-1"),
    );
  });
});
