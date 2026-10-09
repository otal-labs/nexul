import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CloneRoleDialog } from "@/components/settings/CloneRoleDialog";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: () => "Not allowed",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const workspace = (id: string, name: string) => ({ id, name, created_at: "", updated_at: "" });

const role = {
  id: "role-editor",
  workspace_id: "ws-1",
  name: "Editor",
  permissions: ["docs:read"],
  is_owner_role: false,
  created_at: "",
  updated_at: "",
};

const renderDialog = (workspaces: unknown[]) => {
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/workspaces") return { data: workspaces };
    throw new Error(`unexpected GET ${url}`);
  });
  const onClose = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <CloneRoleDialog role={role} open onClose={onClose} />
    </QueryClientProvider>,
  );
  return { onClose };
};

const pickWorkspace = async (name: string) => {
  const user = userEvent.setup();
  await user.click(await screen.findByRole("combobox"));
  await user.click(await screen.findByRole("option", { name }));
  await user.click(screen.getByRole("button", { name: "Clone" }));
};

describe("CloneRoleDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    vi.mocked(toast.success).mockClear();
    vi.mocked(toast.error).mockClear();
  });

  it("offers nothing to submit when the user has no other workspace", async () => {
    const { onClose } = renderDialog([workspace("ws-1", "Engineering")]);

    expect(await screen.findByText("You're not in any other workspace.")).toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clone" })).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalled();
  });

  it("lists the user's other workspaces, never the role's own", async () => {
    renderDialog([workspace("ws-1", "Engineering"), workspace("ws-2", "Marketing"), workspace("ws-3", "Sales")]);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("combobox"));

    expect(await screen.findByRole("option", { name: "Marketing" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Sales" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Engineering" })).not.toBeInTheDocument();
  });

  it("clones into the picked workspace and toasts the result", async () => {
    mocks.post.mockResolvedValue({ data: { ...role, id: "role-2", workspace_id: "ws-2" } });
    const { onClose } = renderDialog([workspace("ws-1", "Engineering"), workspace("ws-2", "Marketing")]);

    await pickWorkspace("Marketing");

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor/clone", { workspace_id: "ws-2" });
    expect(toast.success).toHaveBeenCalledWith("Cloned Editor to Marketing");
  });

  it("names the copy in the toast when the target already had the name", async () => {
    mocks.post.mockResolvedValue({ data: { ...role, id: "role-2", workspace_id: "ws-2", name: "Editor (copy)" } });
    renderDialog([workspace("ws-1", "Engineering"), workspace("ws-2", "Marketing")]);

    await pickWorkspace("Marketing");

    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("Cloned Editor to Marketing as Editor (copy)"));
  });

  it("stays open and toasts the error when the clone is refused", async () => {
    mocks.post.mockRejectedValue(new Error("403"));
    const { onClose } = renderDialog([workspace("ws-1", "Engineering"), workspace("ws-2", "Marketing")]);

    await pickWorkspace("Marketing");

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Not allowed"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("requires a workspace before cloning", async () => {
    renderDialog([workspace("ws-1", "Engineering"), workspace("ws-2", "Marketing")]);

    await userEvent.setup().click(await screen.findByRole("button", { name: "Clone" }));

    expect(await screen.findByText("Pick a workspace")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
