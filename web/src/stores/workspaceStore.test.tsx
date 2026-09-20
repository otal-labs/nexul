import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";

describe("workspaceStore", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "", selectedProjectId: "" });
    useWorkspaceStore.persist.clearStorage();
    localStorage.clear();
  });

  it("starts with no selection", () => {
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("");
  });

  it("selectWorkspace updates the selected id", () => {
    useWorkspaceStore.getState().selectWorkspace("ws-2");
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
  });

  it("selectProject updates the selected id", () => {
    useWorkspaceStore.getState().selectProject("p-2");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("persists selectedWorkspaceId and selectedProjectId, matching sessionStore's convention", () => {
    useWorkspaceStore.getState().selectWorkspace("ws-2");
    useWorkspaceStore.getState().selectProject("p-2");

    const stored = JSON.parse(localStorage.getItem("workspace") ?? "{}");
    expect(stored.state.selectedWorkspaceId).toBe("ws-2");
    expect(stored.state.selectedProjectId).toBe("p-2");
  });
});
