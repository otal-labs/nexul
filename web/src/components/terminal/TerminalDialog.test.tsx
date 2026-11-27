import { act, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TerminalDialog } from "@/components/terminal/TerminalDialog";
import type { TerminalState } from "@/lib/terminal";

const screenState = vi.hoisted(() => ({ report: (() => {}) as (state: TerminalState) => void }));

vi.mock("@/components/terminal/TerminalScreen", () => ({
  TerminalScreen: ({ onState }: { onState: (state: TerminalState) => void }) => {
    screenState.report = onState;
    return null;
  },
}));

describe("TerminalDialog", () => {
  it("closes when the shell exits cleanly", () => {
    const onClose = vi.fn();
    render(<TerminalDialog title="Laptop" socketPath="/pairing/terminals/t1" onClose={onClose} />);
    act(() => screenState.report({ status: "ended", exit: 0 }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("stays open on a failed exit so its code can be read", () => {
    const onClose = vi.fn();
    render(<TerminalDialog title="Laptop" socketPath="/pairing/terminals/t1" onClose={onClose} />);
    act(() => screenState.report({ status: "ended", exit: 127 }));
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("status")).toHaveTextContent("Ended, exit 127");
  });
});
