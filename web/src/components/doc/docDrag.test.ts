// @vitest-environment node
import { describe, expect, it } from "vitest";

import { docDrop } from "@/components/doc/docDrag";
import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";

const folder = (id: string, parentId = ""): DocFolder => ({
  id,
  project_id: "p-1",
  parent_id: parentId,
  name: id,
  is_default: id === "f-main",
  created_at: "",
  updated_at: "",
});

const doc: DocListItem = {
  id: "d-1",
  project_id: "p-1",
  folder_id: "f-main",
  title: "Runbook",
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: "",
  updated_at: "",
};

// Clients holds Acme, which holds Drafts.
const main = folder("f-main");
const clients = folder("f-clients");
const acme = folder("f-acme", "f-clients");
const drafts = folder("f-drafts", "f-acme");
const folders = [main, clients, acme, drafts];

describe("docDrop", () => {
  it("moves a doc into any folder but its own", () => {
    expect(docDrop({ kind: "doc", doc }, drafts, folders)).toEqual({ kind: "doc", id: "d-1", folderId: "f-drafts" });
    expect(docDrop({ kind: "doc", doc }, main, folders)).toBeNull();
  });

  it("moves a folder into any folder outside its own subtree that isn't already its parent", () => {
    expect(docDrop({ kind: "folder", folder: acme }, main, folders)).toEqual({ kind: "folder", id: "f-acme", parentId: "f-main" });
    expect(docDrop({ kind: "folder", folder: acme }, clients, folders)).toBeNull();
    expect(docDrop({ kind: "folder", folder: acme }, acme, folders)).toBeNull();
    expect(docDrop({ kind: "folder", folder: clients }, drafts, folders)).toBeNull();
  });

  it("drops nothing without a held item or a folder under it", () => {
    expect(docDrop(undefined, main, folders)).toBeNull();
    expect(docDrop({ kind: "doc", doc }, undefined, folders)).toBeNull();
  });
});
