import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useHiddenColumns } from "@/hooks/useHiddenColumns";

const box = (left: number, right: number) => () => ({ left, right, top: 0, bottom: 0, width: right - left, height: 0, x: left, y: 0, toJSON: () => ({}) });

const Probe = ({ rights }: { rights: number[] }) => {
  const { ref, hidden } = useHiddenColumns(48);
  return (
    <div
      ref={(node) => {
        if (!node) return;
        node.getBoundingClientRect = box(0, 500) as never;
        ref.current = node;
      }}
    >
      {rights.map((right) => (
        <span
          key={right}
          data-role-head=""
          ref={(node) => {
            if (node) node.getBoundingClientRect = box(right - 96, right) as never;
          }}
        />
      ))}
      <output>{hidden}</output>
    </div>
  );
};

describe("useHiddenColumns", () => {
  it("counts the columns starting past the scroller's edge less its fade, never one partly in view, and none when every column shows", () => {
    const { unmount } = render(<Probe rights={[200, 300, 452, 560, 600]} />);
    expect(screen.getByRole("status").textContent).toBe("2");
    unmount();
    render(<Probe rights={[200, 300, 400]} />);
    expect(screen.getByRole("status").textContent).toBe("0");
  });
});
