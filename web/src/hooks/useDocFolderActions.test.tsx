import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDocFolderActions } from "@/hooks/useDocFolderActions";
import type { DocFolder } from "@/models/DocFolder";

const mocks = vi.hoisted(() => ({ folders: [] as unknown[], confirm: vi.fn(), move: vi.fn() }));

vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@/hooks/useWorkspacePath", () => ({ useWorkspacePath: () => (path: string) => path }));
vi.mock("@/hooks/ProjectHooks", () => ({ useFetchProjects: () => ({ data: [] }) }));
vi.mock("@/hooks/useCreateDocDialog", () => ({ useOpenCreateDoc: () => () => undefined }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);
vi.mock("@/hooks/useFormDialog", () => ({ useFormDialog: () => ({ open: vi.fn() }) }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));
vi.mock("@/hooks/DocFolderHooks", () => ({
  useDeleteDocFolder: () => ({ mutate: vi.fn() }),
  useMoveDocFolder: () => ({ mutate: mocks.move }),
  useFetchDocFolders: () => ({ data: mocks.folders }),
}));

const folder = (id: string, name: string, parentId = "", isDefault = false): DocFolder => ({
  id,
  project_id: "p-1",
  parent_id: parentId,
  name,
  is_default: isDefault,
  created_at: "",
  updated_at: "",
});

const main = folder("f-main", "Main", "", true);
const clients = folder("f-clients", "Clients");
const acme = folder("f-acme", "Acme", "f-clients");
const drafts = folder("f-drafts", "Drafts", "f-acme");

describe("useDocFolderActions", () => {
  it("leaves New doc, Move and Delete off the default folder, which may still hold folders", () => {
    mocks.folders = [main];
    const onMain = renderHook(() => useDocFolderActions(main, 0)).result.current;
    expect(onMain.onNewDoc).toBeUndefined();
    expect(onMain.moveTo).toBeUndefined();
    expect(onMain.onDelete).toBeUndefined();
    expect(onMain.onNewFolder).toBeDefined();
    expect(onMain.onRename).toBeDefined();

    const other = renderHook(() => useDocFolderActions(clients, 0)).result.current;
    expect(other.onNewDoc).toBeDefined();
    expect(other.onDelete).toBeDefined();
  });

  it("moves a folder to the top or into any folder outside its own subtree", () => {
    mocks.folders = [main, clients, acme, drafts];
    const moveTo = renderHook(() => useDocFolderActions(acme, 0)).result.current.moveTo;
    expect(moveTo?.options.map((o) => o.name)).toEqual(["Top level", "Main", "Clients"]);
    expect(moveTo?.currentId).toBe("f-clients");

    moveTo?.onMove("top-level");
    expect(mocks.move).toHaveBeenLastCalledWith({ id: "f-acme", parentId: "" });
    moveTo?.onMove("f-main");
    expect(mocks.move).toHaveBeenLastCalledWith({ id: "f-acme", parentId: "f-main" });
  });

  it("says a delete takes the folders inside and moves every doc to the parent", async () => {
    mocks.folders = [main, clients, acme, drafts];
    mocks.confirm.mockResolvedValue(false);
    const { result } = renderHook(() => useDocFolderActions(acme, 3));
    await act(async () => result.current.onDelete?.());
    expect(mocks.confirm).toHaveBeenLastCalledWith(expect.objectContaining({ message: "The folder inside goes with it, and 3 docs move to Clients. No doc is deleted." }));

    const top = renderHook(() => useDocFolderActions(clients, 0));
    await act(async () => top.result.current.onDelete?.());
    expect(mocks.confirm).toHaveBeenLastCalledWith(expect.objectContaining({ message: "The 2 folders inside go with it. They hold no docs." }));
  });
});
