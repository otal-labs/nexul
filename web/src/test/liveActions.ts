// For a test that isn't about permissions: every action reads as live, as for an Owner.
export const liveActions = {
  useActionState: () => ({ state: "live" }),
  useActionStates: (gates: unknown[]) => gates.map(() => ({ state: "live" })),
};
