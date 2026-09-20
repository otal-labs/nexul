import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
const navigateSpy = vi.hoisted(() => vi.fn());

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => navigateSpy };
});

const workspaces = [
  { id: "ws-1", name: "Engineering", created_at: "", updated_at: "" },
  { id: "ws-2", name: "Marketing", created_at: "", updated_at: "" },
];
const projectsByWorkspace: Record<string, unknown[]> = {
  "ws-1": [{ id: "project-1", name: "Backend", prefix: "BE", position: 0, icon: "", created_at: "", updated_at: "" }],
  "ws-2": [{ id: "project-2", name: "Website", prefix: "WEB", position: 0, icon: "", created_at: "", updated_at: "" }],
};

const renderDialog = (onClose = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <CloneMemoryDialog memoryId="mem-1" open onClose={onClose} />
    </QueryClientProvider>,
  );
  return { onClose };
};

describe("CloneMemoryDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    navigateSpy.mockReset();
    mocks.get.mockImplementation(async (url: string, config?: { params?: { workspace_id?: string } }) => {
      if (url === "/api/workspaces") return { data: workspaces };
      const workspaceId = config?.params?.workspace_id ?? "";
      return { data: projectsByWorkspace[workspaceId] ?? [] };
    });
  });

  it("lists destination projects grouped by workspace, across every workspace the user belongs to", async () => {
    renderDialog();
    await waitFor(() => expect(screen.getByRole("combobox")).toBeInTheDocument());

    const user = userEvent.setup();
    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByRole("option", { name: "Engineering / Workspace" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Engineering / Backend" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Marketing / Workspace" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Marketing / Website" })).toBeInTheDocument();
  });

  it("clones to the selected project and navigates to the clone", async () => {
    mocks.post.mockResolvedValue({ data: { id: "mem-2" } });
    const { onClose } = renderDialog();
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByRole("combobox")).toBeInTheDocument());
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Engineering / Backend" }));
    await user.click(screen.getByRole("button", { name: "Clone" }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/memories/mem-1/clone", {
        project_id: "project-1",
        workspace_id: "ws-1",
      }),
    );
    expect(onClose).toHaveBeenCalled();
    expect(navigateSpy).toHaveBeenCalledWith("/memories/mem-2");
  });

  it("clones to workspace scope and navigates to the clone", async () => {
    mocks.post.mockResolvedValue({ data: { id: "mem-3" } });
    const { onClose } = renderDialog();
    const user = userEvent.setup();

    await waitFor(() => expect(screen.getByRole("combobox")).toBeInTheDocument());
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Marketing / Workspace" }));
    await user.click(screen.getByRole("button", { name: "Clone" }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/memories/mem-1/clone", {
        project_id: "",
        workspace_id: "ws-2",
      }),
    );
    expect(onClose).toHaveBeenCalled();
    expect(navigateSpy).toHaveBeenCalledWith("/memories/mem-3");
  });

  it("cancels without cloning", async () => {
    const { onClose } = renderDialog();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalled();
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
