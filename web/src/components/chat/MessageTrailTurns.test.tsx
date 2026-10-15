import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { MessageTrailTurns } from "@/components/chat/MessageTrailTurns";
import type { ActivityEntry } from "@/models/Trail";
import type { TrailTurn } from "@/utils/ThreadTrailUtility";

const at = (s: number) => `2026-09-18T10:00:${String(s).padStart(2, "0")}Z`;
const read: ActivityEntry = {
  kind: "tool_result",
  call_id: "c-1",
  tool: "Read",
  summary: '{"file_path":"web/src/BoardOrder.ts"}',
  detail: '{"input":{"file_path":"web/src/BoardOrder.ts"},"result":{"content":"export const byStage"}}',
  at: at(1),
};
const failedTest: ActivityEntry = { kind: "tool_result", call_id: "c-2", tool: "Bash", summary: "bun run test · failed", at: at(4) };
const lint: ActivityEntry = { kind: "tool_call", call_id: "c-3", tool: "Bash", summary: "bun run lint", at: at(9) };
const edit: ActivityEntry = { kind: "tool_result", call_id: "c-4", tool: "Edit", summary: "web/src/BoardOrder.ts", at: at(7) };

const turns = (live: ActivityEntry[]): TrailTurn[] => [
  { kind: "turn", entries: [read, failedTest], running: false, from: at(0), until: at(5) },
  { kind: "turn", entries: live, running: true, from: at(5), until: null },
];

describe("MessageTrailTurns", () => {
  it("opens and shuts a finished turn while the next one runs, and keeps it open as new steps arrive", async () => {
    const user = userEvent.setup();
    const { rerender } = render(<MessageTrailTurns turns={turns([edit])} />);
    const finished = screen.getByRole("button", { name: /^Worked for 5s · explored 1 file, ran 1 command/ });
    expect(screen.queryByText("Explored 1 file")).not.toBeInTheDocument();

    await user.click(finished);
    expect(screen.getByText("Explored 1 file")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();

    rerender(<MessageTrailTurns turns={turns([edit, lint])} />);
    expect(screen.getByRole("button", { name: /^Running bun run lint/ })).toBeInTheDocument();
    expect(screen.getByText("Explored 1 file")).toBeInTheDocument();

    await user.click(finished);
    expect(screen.queryByText("Explored 1 file")).not.toBeInTheDocument();
  });

  it("opens a file's arguments and result from its name, and shuts them from it again", async () => {
    const user = userEvent.setup();
    render(<MessageTrailTurns turns={turns([edit])} />);
    await user.click(screen.getByRole("button", { name: /^Worked for 5s/ }));

    const file = screen.getByRole("button", { name: "BoardOrder.ts" });
    await user.click(file);
    expect(file).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("export const byStage")).toBeVisible();

    await user.click(file);
    expect(screen.queryByText("export const byStage")).not.toBeInTheDocument();
  });
});
