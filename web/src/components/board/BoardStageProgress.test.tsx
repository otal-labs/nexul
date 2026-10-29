import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { BoardStageProgress } from "@/components/board/BoardStageProgress";
import type { BoardStatus } from "@/models/Status";

const column = (id: string, kind: BoardStatus["kind"], position: number): BoardStatus => ({
  id,
  name: id,
  position,
  kind,
  icon: "",
  created_at: "",
  updated_at: "",
});

const columns = [column("open", "backlog", 0), column("doing", "progress", 1), column("done", "done", 2)];

describe("BoardStageProgress", () => {
  it("draws a segment per stage, as wide as its share", () => {
    const { container } = render(<BoardStageProgress columns={columns} counts={{ open: 4, doing: 1 }} />);
    const segments = [...container.querySelectorAll("[data-stage-segment]")];
    expect(segments.map((segment) => segment.getAttribute("data-stage-segment"))).toEqual(["open", "doing", "done"]);
    expect((segments[0] as HTMLElement).style.flexGrow).toBe("4");
    expect((segments[1] as HTMLElement).style.flexGrow).toBe("1");
  });

  it("keeps a stage with nothing on show as a sliver, so the bar still matches the stage row", () => {
    const { container } = render(<BoardStageProgress columns={columns} counts={{ open: 4, done: 0 }} />);
    const empty = container.querySelector('[data-stage-segment="done"]');
    expect(empty).not.toBeNull();
    expect((empty as HTMLElement).style.flexGrow).toBe("1");
    expect(empty?.className).toContain("opacity-25");
  });

  it("draws nothing when the board shows no tickets", () => {
    const { container } = render(<BoardStageProgress columns={columns} counts={{ open: 0, doing: 0, done: 0 }} />);
    expect(container.querySelector("[data-stage-segment]")).toBeNull();
  });
});
