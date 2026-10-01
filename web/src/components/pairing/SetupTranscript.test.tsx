import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { SetupTranscript } from "@/components/pairing/SetupTranscript";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import type { SetupRunRow } from "@/models/Pairing";
import type { ActivityEntry } from "@/models/Trail";

const row = (state: SetupRunRow["state"]): SetupRunRow => ({ provider: "codex", name: "Codex", state, status: "Confirmed with 119 skills", model: "", turnId: "t1" });

const said = "Nexul's MCP tools are available, and the installed nexul-memory version matches. I'll finish checking the skills.";
const listed = '/bin/bash -lc "ls ~/.claude/skills"';
const running = "/bin/bash -lc \"python3 - <<'PY'\"";

// A Codex setup turn as the live frames leave it: its messages, a command and an MCP tool, then a command still open.
const steps: ActivityEntry[] = [
  { kind: "text", call_id: "", tool: "", summary: "I'll check both skill folders.", detail: "I'll check both skill folders.", at: "2026-10-01T10:28:57Z" },
  { kind: "tool_result", call_id: "c1", tool: "Shell", summary: listed, detail: "", at: "2026-10-01T10:29:01Z" },
  { kind: "tool_result", call_id: "c2", tool: "nexul · skill_get", summary: "nexul · skill_get", detail: "", at: "2026-10-01T10:29:18Z" },
  { kind: "text", call_id: "", tool: "", summary: "Nexul's MCP tools are available, and the…", detail: said, at: "2026-10-01T10:29:40Z" },
  { kind: "tool_call", call_id: "c3", tool: "Shell", summary: running, detail: "", at: "2026-10-01T10:30:04Z" },
];

const renderTranscript = (state: SetupRunRow["state"]) =>
  render(<SetupTranscript row={row(state)} runningName={undefined} retryDisabled={false} onRetry={() => {}} />);

describe("SetupTranscript", () => {
  beforeEach(() => useSetupActivityStore.setState({ steps: { t1: steps } }));

  it("reads the agent's messages whole as prose and folds the tools between them into counted groups, the live one open", () => {
    renderTranscript("running");

    expect(screen.getByText(said)).toBeInTheDocument();
    expect(screen.getByText("Used 1 tool and ran 1 command")).toBeInTheDocument();
    expect(screen.queryByText(listed)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Working/ })).toBeInTheDocument();
    expect(screen.getByText(running)).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "running" })).toBeInTheDocument();
  });

  it("folds the live group shut once the turn is confirmed, leaving its count and the outcome", () => {
    const { rerender } = renderTranscript("running");

    rerender(<SetupTranscript row={row("confirmed")} runningName={undefined} retryDisabled={false} onRetry={() => {}} />);

    expect(screen.queryByText(running)).not.toBeInTheDocument();
    expect(screen.getByText("Ran 1 command")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Working/ })).not.toBeInTheDocument();
    expect(screen.getByText(/Confirmed with 119 skills/)).toBeInTheDocument();
  });
});
