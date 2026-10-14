import { actionState, type ActionGate } from "@nexul/client-core/actionState";

// What the mocked action state reads, in place of the viewer's /me. Reset it in beforeEach.
export const viewer = {
  workspace: [] as string[],
  instance: [] as string[],
  instanceOwner: false,
  showLocked: false,
};

export const resetViewer = () => {
  viewer.workspace = [];
  viewer.instance = [];
  viewer.instanceOwner = false;
  viewer.showLocked = false;
};

// Runs the real actionState on the viewer above: `vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState)`.
export const mockedActionState = {
  useActionState: (gate: ActionGate) =>
    actionState({
      ...gate,
      me: { role_name: "Member", permissions: viewer.workspace, show_locked_actions: viewer.showLocked },
      instance: { instance_permissions: viewer.instance, instance_owner: viewer.instanceOwner },
    }),
};
