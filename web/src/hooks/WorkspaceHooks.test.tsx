import type { Location } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { getArchivedWorkspacesKey, workspaceFollower } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Workspace } from "@/models/Workspace";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() }) }));

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

describe("the workspace follower on archive and restore", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  });

  const frame = (id: string) => ({ workspace_id: id, name: "Acme", slug: "acme", user_ids: ["u-1"] });
  const lists = () =>
    seeded([
      [["getWorkspaces"], [workspace("ws-1", "acme", "Acme"), workspace("ws-2", "other", "Other")]],
      [[getArchivedWorkspacesKey], []],
    ]);

  it("takes the open workspace off the list and moves the page on before its address can render not found", async () => {
    const client = lists();
    const { navigate } = await followFrame(workspaceFollower, "workspace.archived", frame("ws-1"), client, at("/acme/board"));
    expect(client.getQueryData<Workspace[]>(["getWorkspaces"])?.map((w) => w.id)).toEqual(["ws-2"]);
    expect(navigate).toHaveBeenCalledWith("/", { replace: true });
    expect(isStale(client, [getArchivedWorkspacesKey])).toBe(true);
  });

  it("only refreshes the lists when the archived workspace is not the one on screen", async () => {
    const client = lists();
    const { navigate } = await followFrame(workspaceFollower, "workspace.archived", frame("ws-2"), client, at("/acme/board"));
    expect(navigate).not.toHaveBeenCalled();
    expect(client.getQueryData<Workspace[]>(["getWorkspaces"])).toHaveLength(2);
    expect(isStale(client, ["getWorkspaces"])).toBe(true);
  });

  it("brings a restored workspace back by refreshing both lists", async () => {
    const client = lists();
    await followFrame(workspaceFollower, "workspace.restored", frame("ws-2"), client);
    expect(isStale(client, ["getWorkspaces"])).toBe(true);
    expect(isStale(client, [getArchivedWorkspacesKey])).toBe(true);
  });
});
