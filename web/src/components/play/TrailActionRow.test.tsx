import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { TrailActionRow } from "@/components/play/TrailActionRow";
import type { ActivityEntry } from "@/models/Trail";

const entry = (overrides: Partial<ActivityEntry>): ActivityEntry => ({
  kind: "tool_call",
  call_id: "c-1",
  tool: "Read",
  summary: '{"file_path":"main.go"}',
  at: "2026-09-18T10:00:00Z",
  ...overrides,
});

const renderRow = (e: ActivityEntry, live = false) =>
  render(
    <ul>
      <TrailActionRow entry={e} live={live} />
    </ul>,
  );

describe("TrailActionRow", () => {
  it("a tool call reads `tool: args` in one line under a wrench, and spins while live", () => {
    renderRow(entry({}), true);
    expect(screen.getByRole("img", { name: "tool call" })).toBeInTheDocument();
    expect(screen.getByText('Read: {"file_path":"main.go"}')).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "running" })).toBeInTheDocument();
  });

  it("a tool call in a finished trail sits as a dot, not a spinner", () => {
    renderRow(entry({}));
    expect(screen.queryByRole("img", { name: "running" })).not.toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "done" })).not.toBeInTheDocument();
  });

  it("a command row shows the command alone under a terminal icon", () => {
    renderRow(entry({ tool: "Bash", summary: "go test ./..." }));
    expect(screen.getByRole("img", { name: "command" })).toBeInTheDocument();
    expect(screen.getByText("go test ./...")).toBeInTheDocument();
    expect(screen.queryByText(/Bash/)).not.toBeInTheDocument();
  });

  it("a file change row names the path under a file icon", () => {
    renderRow(entry({ kind: "tool_result", tool: "Edit", summary: "/home/dev/Code/nexul/.golangci.yml" }));
    expect(screen.getByRole("img", { name: "file change" })).toBeInTheDocument();
    expect(screen.getByText("Edit: /home/dev/Code/nexul/.golangci.yml")).toBeInTheDocument();
  });

  it("a tool result gets a check, a result preview, and expands to its arguments and result", async () => {
    const user = userEvent.setup();
    renderRow(
      entry({
        kind: "tool_result",
        detail: '{"input":{"file_path":"main.go"},"result":{"type":"tool_result","content":"package main\\n\\nfunc main() {}"}}',
      }),
    );
    expect(screen.getByRole("img", { name: "tool result" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "done" })).toBeInTheDocument();
    expect(screen.getByText("→")).toBeInTheDocument();
    expect(screen.getByText("package main func main() {}", { selector: "span" })).toBeInTheDocument();

    expect(screen.getByText("Arguments")).not.toBeVisible();
    await user.click(screen.getByText('Read: {"file_path":"main.go"}'));
    expect(screen.getByText("Arguments")).toBeVisible();
    expect(screen.getByText('{ "file_path": "main.go" }', { normalizer: (s) => s.replace(/\s+/g, " ").trim() })).toBeInTheDocument();
    expect(screen.getByText("Result")).toBeInTheDocument();
    expect(screen.getByText("package main func main() {}", { selector: "pre", normalizer: (s) => s.replace(/\s+/g, " ").trim() })).toBeInTheDocument();
  });

  it("a failed result keeps the label suffix and shows a cross instead of a check", () => {
    renderRow(entry({ kind: "tool_result", tool: "Bash", summary: "go test · failed", detail: '{"input":{"command":"go test"}}' }));
    expect(screen.getByText("go test · failed")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "failed" })).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "done" })).not.toBeInTheDocument();
  });

  it("the Agent's own sentence is a reasoning row that expands to the full text", async () => {
    const user = userEvent.setup();
    renderRow({ kind: "text", summary: "Opened PR #7", detail: "Opened PR #7\n\nAll tests pass.", at: "2026-09-18T10:00:09Z" });
    expect(screen.getByRole("img", { name: "reasoning" })).toBeInTheDocument();
    expect(screen.getByText("Opened PR #7")).toBeInTheDocument();
    expect(screen.getByText(/All tests pass\./)).not.toBeVisible();
    await user.click(screen.getByText("Opened PR #7"));
    expect(screen.getByText(/All tests pass\./)).toBeVisible();
  });

  it("a question shows the question mark and a legacy line shows a plain dot with no expander", () => {
    renderRow(entry({ kind: "question", tool: "AskUserQuestion", summary: '{"questions":[]}' }), true);
    expect(screen.getByRole("img", { name: "question" })).toBeInTheDocument();

    renderRow({ kind: "other", summary: "Read main.go", at: "" });
    expect(screen.getByRole("img", { name: "step" })).toBeInTheDocument();
    expect(screen.getByText("Read main.go")).toBeInTheDocument();
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
  });
});
