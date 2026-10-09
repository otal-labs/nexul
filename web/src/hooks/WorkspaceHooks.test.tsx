import type { Location } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import { workspaceFollower } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Workspace } from "@/models/Workspace";
import { followFrame, seeded } from "@/test/followFrame";

const workspace = (id: string, slug: string, name: string): Workspace => ({
  id,
  slug,
  name,
  mention_chip_template: "",
  mention_chip_template_edited: false,
  created_at: "",
  updated_at: "",
});

const at = (pathname: string, search = "", hash = ""): Location => ({ pathname, search, hash, state: null, key: "default" });

describe("the workspace follower", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  });

  const lists = () => seeded([[["getWorkspaces"], [workspace("ws-1", "acme", "Acme"), workspace("ws-2", "other", "Other")]]]);

  it("moves the open page onto the workspace's new slug, keeping the rest of the address, and renames it in the list", async () => {
    const client = lists();
    const update = { workspace_id: "ws-1", name: "Acme Labs", slug: "acme-labs" };
    const { navigate } = await followFrame(workspaceFollower, "workspace.updated", update, client, at("/acme/configuration/general", "?tab=x", "#top"));
    expect(navigate).toHaveBeenCalledWith("/acme-labs/configuration/general?tab=x#top", { replace: true });
    expect(useWorkspaceStore.getState().selectedWorkspaceSlug).toBe("acme-labs");
    expect(client.getQueryData<Workspace[]>(["getWorkspaces"])?.map((w) => [w.slug, w.name])).toEqual([
      ["acme-labs", "Acme Labs"],
      ["other", "Other"],
    ]);
  });

  it("leaves the address alone when the renamed workspace is not the one on screen", async () => {
    const client = lists();
    const { navigate } = await followFrame(workspaceFollower, "workspace.updated", { workspace_id: "ws-2", name: "Other Co", slug: "other-co" }, client, at("/acme/board"));
    expect(navigate).not.toHaveBeenCalled();
    expect(client.getQueryData<Workspace[]>(["getWorkspaces"])?.[1]?.slug).toBe("other-co");
  });
});
