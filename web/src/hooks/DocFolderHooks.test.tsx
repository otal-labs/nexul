import { describe, expect, it } from "vitest";

import { docFolderFollower } from "@/hooks/DocFolderHooks";
import type { DocFolder } from "@/models/DocFolder";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const specs: DocFolder = { id: "f-1", project_id: "p-1", name: "Specs", is_default: false, created_at: "", updated_at: "" };

describe("the doc folder follower", () => {
  it("renames and drops folders in their project's list, and refetches only that list for a new one", async () => {
    const client = seeded([
      [["getDocFolders", "p-1"], [specs]],
      [["getDocFolders", "p-2"], []],
    ]);
    await followFrame(docFolderFollower, "doc.folder.updated", { folder: { ...specs, name: "Plans" } }, client);
    expect(client.getQueryData<DocFolder[]>(["getDocFolders", "p-1"])?.[0]?.name).toBe("Plans");

    await followFrame(docFolderFollower, "doc.folder.created", { folder: { ...specs, id: "f-2" } }, client);
    expect([isStale(client, ["getDocFolders", "p-1"]), isStale(client, ["getDocFolders", "p-2"])]).toEqual([true, false]);

    await followFrame(docFolderFollower, "doc.folder.deleted", { folder: specs, moved_to_folder_id: "f-0" }, client);
    expect(client.getQueryData(["getDocFolders", "p-1"])).toEqual([]);
  });
});
