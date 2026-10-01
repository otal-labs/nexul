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
    renderRow(entry({ tool: "WebFetch", summary: '{"url":"https://nexul.io"}' }), true);
    expect(screen.getByRole("img", { name: "tool call" })).toBeInTheDocument();
    expect(screen.getByText('WebFetch: {"url":"https://nexul.io"}')).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "running" })).toBeInTheDocument();
  });

  it("a tool call in a finished trail neither spins nor carries a check mark", () => {
    renderRow(entry({}));
    expect(screen.queryByRole("img", { name: "running" })).not.toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "done" })).not.toBeInTheDocument();
  });

  it("an MCP call reads as its server and tool under a wrench", () => {
    renderRow(entry({ kind: "tool_result", tool: "mcp__nexul__skill_get", summary: '{"name":"nexul-memory"}' }));
    expect(screen.getByRole("img", { name: "tool result" })).toBeInTheDocument();
    expect(screen.getByText("Nexul · skill_get")).toBeInTheDocument();
  });

  it("a command row shows the command alone, out of its shell wrapper, under a terminal icon", () => {
    renderRow(entry({ tool: "Shell", summary: `/bin/bash -lc "go test ./..."` }));
    expect(screen.getByRole("img", { name: "command" })).toBeInTheDocument();
    expect(screen.getByText("go test ./...")).toBeInTheDocument();
    expect(screen.queryByText(/bash|Shell/)).not.toBeInTheDocument();
  });

  it("a file change row names the path under a file icon", () => {
    renderRow(entry({ kind: "tool_result", tool: "Edit", summary: "/home/dev/Code/nexul/.golangci.yml" }));
    expect(screen.getByRole("img", { name: "file change" })).toBeInTheDocument();
    expect(screen.getByText("Edit: /home/dev/Code/nexul/.golangci.yml")).toBeInTheDocument();
  });

  it("a file read names its path under a file icon", () => {
    renderRow(entry({ kind: "tool_result" }));
    expect(screen.getByRole("img", { name: "file" })).toBeInTheDocument();
    expect(screen.getByText("Read: main.go")).toBeInTheDocument();
  });

  it("a finished row is one line with no time, check, or result preview, and expands to its arguments, result and time", async () => {
    const user = userEvent.setup();
    renderRow(
      entry({
        kind: "tool_result",
        detail: '{"input":{"file_path":"main.go"},"result":{"type":"tool_result","content":"package main\\n\\nfunc main() {}"}}',
      }),
    );
    const row = screen.getByRole("listitem");
    expect(row).toHaveTextContent(/^Read: main\.go/);
    expect(screen.queryByRole("img", { name: "done" })).not.toBeInTheDocument();
    expect(screen.queryByText("package main func main() {}", { selector: "span" })).not.toBeInTheDocument();
    const time = new Date("2026-09-18T10:00:00Z").toLocaleTimeString([], { hour12: false });
    expect(screen.getByText(time)).not.toBeVisible();

    await user.click(screen.getByText("Read: main.go"));
    expect(screen.getByText(time)).toBeVisible();
    expect(screen.getByText("Arguments")).toBeVisible();
    expect(screen.getByText('{ "file_path": "main.go" }', { normalizer: (s) => s.replace(/\s+/g, " ").trim() })).toBeInTheDocument();
    expect(screen.getByText("package main func main() {}", { selector: "pre", normalizer: (s) => s.replace(/\s+/g, " ").trim() })).toBeInTheDocument();
  });

  it("a row with nothing to expand has no expander", () => {
    renderRow(entry({ kind: "tool_result" }));
    expect(screen.getByText("Read: main.go")).toBeInTheDocument();
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
  });

  it("a failed command marks its icon failed, drops the marker from the label, and shows the error when expanded", async () => {
    const user = userEvent.setup();
    renderRow(
      entry({
        kind: "tool_result",
        tool: "Bash",
        summary: "go test · failed",
        detail: '{"input":{"command":"go test"},"result":{"content":"FAIL nexul/internal/plays"}}',
      }),
    );
    expect(screen.getByRole("img", { name: "failed" })).toBeInTheDocument();
    expect(screen.getByText("go test")).toBeInTheDocument();
    expect(screen.queryByText(/· failed/)).not.toBeInTheDocument();
    await user.click(screen.getByText("go test"));
    expect(screen.getByText("FAIL nexul/internal/plays")).toBeVisible();
  });

  it("the Agent's own sentence reads as prose with no icon and expands to the full text", async () => {
    const user = userEvent.setup();
    renderRow({ kind: "text", summary: "Opened PR #7", detail: "Opened PR #7\n\nAll tests pass.", at: "2026-09-18T10:00:09Z" });
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
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
