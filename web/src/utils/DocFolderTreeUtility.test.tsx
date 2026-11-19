// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { DocFolder } from "@/models/DocFolder";
import { folderOptions, folderPath, folderTree, subtreeIds } from "@/utils/DocFolderTreeUtility";

const folder = (id: string, name: string, parentId = ""): DocFolder => ({
  id,
  project_id: "p-1",
  parent_id: parentId,
  name,
  is_default: id === "f-main",
  created_at: "",
  updated_at: "",
});

// Listed in creation order, as the server sends them: Drafts was made before the folder it sits in moved under Clients.
const folders = [folder("f-main", "Main"), folder("f-drafts", "Drafts", "f-acme"), folder("f-clients", "Clients"), folder("f-acme", "Acme", "f-clients")];

describe("DocFolderTreeUtility", () => {
  it("walks the tree depth first, each folder right after the one it sits in", () => {
    expect(folderTree(folders).map(({ folder: f, depth }) => [f.name, depth])).toEqual([
      ["Main", 0],
      ["Clients", 0],
      ["Acme", 1],
      ["Drafts", 2],
    ]);
  });

  it("puts a folder whose parent isn't listed at the top", () => {
    expect(folderTree([folder("f-orphan", "Orphan", "f-hidden")]).map(({ depth }) => depth)).toEqual([0]);
  });

  it("names a folder by its path and picks list every folder that way", () => {
    expect(folderPath(folders, "f-drafts")).toBe("Clients / Acme / Drafts");
    expect(folderPath(folders, "f-gone")).toBe("");
    expect(folderOptions(folders).map((o) => o.name)).toEqual(["Main", "Clients", "Clients / Acme", "Clients / Acme / Drafts"]);
  });

  it("finds a folder's subtree", () => {
    expect([...subtreeIds(folders, "f-clients")].sort()).toEqual(["f-acme", "f-clients", "f-drafts"]);
    expect([...subtreeIds(folders, "f-drafts")]).toEqual(["f-drafts"]);
  });
});
