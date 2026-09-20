import { render, screen } from "@testing-library/react";
import { act } from "react";
import { applyAwarenessUpdate, Awareness, encodeAwarenessUpdate } from "y-protocols/awareness";
import { describe, expect, it } from "vitest";
import * as Y from "yjs";

import { PointerOverlay } from "@/components/doc/collab/PointerOverlay";

const remoteState = (state: Record<string, unknown>) => {
  const doc = new Y.Doc();
  const awareness = new Awareness(doc);
  awareness.setLocalState(state);
  return { clientID: doc.clientID, update: encodeAwarenessUpdate(awareness, [doc.clientID]) };
};

describe("PointerOverlay", () => {
  it("renders a named colored pointer per remote participant with a position", async () => {
    const doc = new Y.Doc();
    const awareness = new Awareness(doc);
    render(<PointerOverlay awareness={awareness} selfID={doc.clientID} />);

    const bob = remoteState({ user: { name: "Bob", color: "#f00" }, pointer: { x: 40, y: 80 } });
    await act(async () => {
      applyAwarenessUpdate(awareness, bob.update, "remote");
    });

    const pointer = screen.getByTestId("remote-pointer");
    expect(pointer).toHaveTextContent("Bob");
    expect(pointer).toHaveStyle({ transform: "translate(40px, 80px)" });
  });

  it("hides participants without a pointer and never renders itself", async () => {
    const doc = new Y.Doc();
    const awareness = new Awareness(doc);
    awareness.setLocalState({ user: { name: "Me", color: "#00f" }, pointer: { x: 1, y: 1 } });
    render(<PointerOverlay awareness={awareness} selfID={doc.clientID} />);

    const idle = remoteState({ user: { name: "Idle", color: "#0f0" }, pointer: null });
    await act(async () => {
      applyAwarenessUpdate(awareness, idle.update, "remote");
    });

    expect(screen.queryByTestId("remote-pointer")).not.toBeInTheDocument();
  });
});
