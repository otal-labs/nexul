import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PlayHealthLine } from "@/components/play/PlayHealthLine";
import type { PlayRunHealth } from "@/models/Play";

const health = (overrides: Partial<PlayRunHealth> = {}): PlayRunHealth => ({
  last_run: { state: "done", started_at: new Date(Date.now() - 2 * 3600_000).toISOString() },
  runs_7d: 11,
  failed_7d: 0,
  waiting: 0,
  running: 0,
  ...overrides,
});

const line = (h: PlayRunHealth) => render(<PlayHealthLine health={h} />).container.textContent;

describe("PlayHealthLine", () => {
  it.each([
    ["a play that never ran", health({ last_run: null, runs_7d: 0 }), "Not run yet"],
    ["a quiet play, its week and its last run", health(), /^11 runs this week.*last .*ago$/],
    ["a busy play, busiest first and no last run while one is in flight", health({ running: 1, waiting: 2, failed_7d: 1 }), "1 running2 need answers1 failed this week11 runs this week"],
    ["one run waiting, in the singular", health({ waiting: 1, runs_7d: 1 }), /^1 needs an answer1 run this week/],
  ])("reads %s", (_, h, text) => {
    expect(line(h)).toMatch(text);
  });
});
