import { describe, expect, it } from "vitest";

import { groupDocs, type DocFolderGroup, type DocListGroups } from "@/components/doc/docGroups";
import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";

const folder = (id: string, name: string, isDefault = false): DocFolder => ({
  id,
  project_id: "project-1",
  parent_id: "",
  name,
  is_default: isDefault,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
});

const doc = (id: string, folderId: string, created: string, updated = created): DocListItem => ({
  id,
  project_id: "project-1",
  folder_id: folderId,
  title: id,
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: created,
  updated_at: updated,
});

const main = folder("f-main", "Main", true);
const getSource = folder("f-gs", "GetSource");
const empty = folder("f-empty", "Empty");
const folders = [main, getSource, empty];

// EP01 was made first and edited last; EP02 and the roadmap were left alone after they were made.
const docs = [
  doc("roadmap", "f-main", "2026-01-02T09:00:00Z"),
  doc("EP01", "f-gs", "2026-01-01T09:00:00Z", "2026-01-09T09:00:00Z"),
  doc("EP02", "f-gs", "2026-01-03T09:00:00Z"),
];

const shape = (groups: DocListGroups) => ({
  pinned: groups.pinned.map((d) => d.id),
  folders: groups.folders.map((g) => [g.folder.name, g.docs.map((d) => d.id), g.total]),
});

describe("groupDocs", () => {
  it("groups docs under their folders in the order given, keeping empty folders", () => {
    expect(shape(groupDocs({ docs, folders, sortBy: "created_at", pinnedIds: [] }))).toEqual({
      pinned: [],
      folders: [
        ["Main", ["roadmap"], 1],
        ["GetSource", ["EP02", "EP01"], 2],
        ["Empty", [], 0],
      ],
    });
  });

  it("orders a folder's docs newest first by the chosen timestamp", () => {
    const groups = groupDocs({ docs, folders, sortBy: "updated_at", pinnedIds: [] });
    expect(groups.folders[1]?.docs.map((d) => d.id)).toEqual(["EP01", "EP02"]);
  });

  it("lifts pinned docs into Pinned in pin order, still counting them in their folder", () => {
    expect(shape(groupDocs({ docs, folders, sortBy: "created_at", pinnedIds: ["gone", "EP01", "roadmap"] }))).toEqual({
      pinned: ["EP01", "roadmap"],
      folders: [
        ["Main", [], 1],
        ["GetSource", ["EP02"], 2],
        ["Empty", [], 0],
      ],
    });
  });

  it("files a doc whose folder is not listed under the default folder", () => {
    const stray = doc("stray", "f-unknown", "2026-01-04T09:00:00Z");
    const groups = groupDocs({ docs: [...docs, stray], folders, sortBy: "created_at", pinnedIds: [] });
    expect(groups.folders[0]?.docs.map((d) => d.id)).toEqual(["stray", "roadmap"]);
  });

  it("while searching, hides folders without a match and keeps every doc in a folder's count", () => {
    const groups = groupDocs({ docs, folders, sortBy: "created_at", pinnedIds: ["roadmap"], match: (d) => d.id.startsWith("EP0") });
    expect(shape(groups)).toEqual({ pinned: [], folders: [["GetSource", ["EP02", "EP01"], 2]] });
  });
});

describe("groupDocs with folders inside folders", () => {
  const clients = { ...folder("f-clients", "Clients") };
  const acme = { ...folder("f-acme", "Acme"), parent_id: "f-clients" };
  const drafts = { ...folder("f-drafts", "Drafts"), parent_id: "f-acme" };
  const nested = [main, acme, clients, drafts];
  const nestedDocs = [doc("brief", "f-acme", "2026-01-02T09:00:00Z"), doc("draft", "f-drafts", "2026-01-03T09:00:00Z"), doc("roadmap", "f-main", "2026-01-01T09:00:00Z")];
  const tree = (groups: DocFolderGroup[]): unknown[] => groups.map((g) => [g.folder.name, g.docs.map((d) => d.id), g.total, tree(g.folders)]);

  it("puts each folder under its parent and counts the docs of every folder inside", () => {
    const groups = groupDocs({ docs: nestedDocs, folders: nested, sortBy: "created_at", pinnedIds: [] });
    expect(tree(groups.folders)).toEqual([
      ["Main", ["roadmap"], 1, []],
      ["Clients", [], 2, [["Acme", ["brief"], 2, [["Drafts", ["draft"], 1, []]]]]],
    ]);
  });

  it("while searching, keeps the folders on the way to a match and drops the rest", () => {
    const groups = groupDocs({ docs: nestedDocs, folders: nested, sortBy: "created_at", pinnedIds: [], match: (d) => d.id === "draft" });
    expect(tree(groups.folders)).toEqual([["Clients", [], 2, [["Acme", [], 2, [["Drafts", ["draft"], 1, []]]]]]]);
  });
});
