import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDocFolderActions } from "@/hooks/useDocFolderActions";
import type { DocFolder } from "@/models/DocFolder";

vi.mock("@/hooks/useCreateDocDialog", () => ({ useCreateDocDialog: () => () => undefined }));
vi.mock("@/hooks/WorkspaceHooks", () => ({ useHasPermission: () => true }));
vi.mock("@/hooks/useFormDialog", () => ({ useFormDialog: () => ({ open: vi.fn() }) }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: vi.fn() }) }));
vi.mock("@/hooks/DocFolderHooks", () => ({ useDeleteDocFolder: () => ({ mutate: vi.fn() }), useFetchDocFolders: () => ({ data: [] }) }));

const folder = (isDefault: boolean): DocFolder => ({ id: "f", project_id: "p-1", name: "F", is_default: isDefault, created_at: "", updated_at: "" });

describe("useDocFolderActions", () => {
  it("leaves New doc off the default folder, where the pane's own + already files docs", () => {
    const main = renderHook(() => useDocFolderActions(folder(true), 0)).result.current;
    expect(main.onNewDoc).toBeUndefined();
    expect(main.onRename).toBeDefined();
    expect(main.onDelete).toBeUndefined();

    const other = renderHook(() => useDocFolderActions(folder(false), 0)).result.current;
    expect(other.onNewDoc).toBeDefined();
    expect(other.onDelete).toBeDefined();
  });
});
