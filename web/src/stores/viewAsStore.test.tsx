import { beforeEach, describe, expect, it } from "vitest";

import { useViewAsStore } from "@/stores/viewAsStore";

const alice = { workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "" };

describe("viewAsStore", () => {
  beforeEach(() => {
    useViewAsStore.getState().exit();
    sessionStorage.clear();
    localStorage.clear();
  });

  it("keeps a view in this tab's session storage and never in storage another tab shares", () => {
    useViewAsStore.getState().start(alice);
    expect(JSON.parse(sessionStorage.getItem("view-as") ?? "{}").state.target).toEqual(alice);
    expect(localStorage.getItem("view-as")).toBeNull();
  });
});
