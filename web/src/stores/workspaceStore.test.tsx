import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";

describe("workspaceStore", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "", selectedProjectId: "" });
    useWorkspaceStore.persist.clearStorage();
    localStorage.clear();
  });

  it("persists the selected workspace and project", () => {
    useWorkspaceStore.getState().selectWorkspace("ws-2", "otal");
    useWorkspaceStore.getState().selectProject("p-2");

    const stored = JSON.parse(localStorage.getItem("workspace") ?? "{}");
    expect(stored.state.selectedWorkspaceId).toBe("ws-2");
    expect(stored.state.selectedWorkspaceSlug).toBe("otal");
    expect(stored.state.selectedProjectId).toBe("p-2");
  });
});
