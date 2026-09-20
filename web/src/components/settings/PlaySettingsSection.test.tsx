import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play = (overrides: Record<string, unknown> = {}) => ({
  id: "play-fix",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "Reads the ticket and opens a PR.",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  created_by: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const mockPlaysAndProjects = (plays: unknown[]) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/workspaces/ws-1/plays") return Promise.resolve({ data: plays });
    if (url === "/api/projects") return Promise.resolve({ data: [] });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
};

const renderSection = (props: { canWrite?: boolean; canDelete?: boolean } = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <PlaySettingsSection canWrite={props.canWrite ?? true} canDelete={props.canDelete ?? true} />
    </QueryClientProvider>,
  );
};

describe("PlaySettingsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.delete.mockReset();
    mocks.errorMessage.mockClear();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("shows an error when the play list fails to load", async () => {
    mocks.errorMessage.mockReturnValue("Plays failed");
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/projects") return Promise.resolve({ data: [] });
      return Promise.reject(new Error("boom"));
    });
    renderSection();
    expect(await screen.findByText("Plays failed")).toBeInTheDocument();
  });

  it("shows an empty state with no plays", async () => {
    mockPlaysAndProjects([]);
    renderSection();
    expect(await screen.findByText("No plays yet")).toBeInTheDocument();
  });

  it("lists a play with its type badge and stage", async () => {
    mockPlaysAndProjects([play()]);
    renderSection();
    expect(await screen.findByText("Fix with AI")).toBeInTheDocument();
    expect(screen.getByText("Ticket")).toBeInTheDocument();
    expect(screen.getByText("In progress")).toBeInTheDocument();
  });

  it("marks a disabled play", async () => {
    mockPlaysAndProjects([play({ enabled: false })]);
    renderSection();
    expect(await screen.findByText("Disabled")).toBeInTheDocument();
  });

  it("hides the New play button without plays:write", async () => {
    mockPlaysAndProjects([play()]);
    renderSection({ canWrite: false });
    await screen.findByText("Fix with AI");
    expect(screen.queryByRole("button", { name: "New play" })).not.toBeInTheDocument();
  });

  it("hides edit and delete without their permissions", async () => {
    mockPlaysAndProjects([play()]);
    renderSection({ canWrite: false, canDelete: false });
    await screen.findByText("Fix with AI");
    expect(screen.queryByRole("button", { name: /edit play/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete play/i })).not.toBeInTheDocument();
  });

  it("deletes a play after confirming", async () => {
    mockPlaysAndProjects([play()]);
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Delete play Fix with AI" }));
    await user.click(screen.getByRole("button", { name: /^confirm$/i }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix");
  });
});
