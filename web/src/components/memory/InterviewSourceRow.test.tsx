import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InterviewSourceRow } from "@/components/memory/InterviewSourceRow";
import type { InterviewSource } from "@/models/InterviewSource";

const mocks = vi.hoisted(() => ({ patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: (e: Error) => e.message }));
vi.mock("@/hooks/useActionState", () => ({ useActionState: () => ({ state: "live" }) }));

const source = (over: Partial<InterviewSource>): InterviewSource => ({
  id: "s-1", workspace_id: "w-1", project_id: "p-1", kind: "text", ref: "", label: "Huly", body: "Old notes", stance: "question",
  added_by: "u-1", added_at: "2026-10-01T00:00:00Z", updated_at: "2026-10-01T00:00:00Z", ...over,
});

const renderRow = (s: InterviewSource) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ul>
        <InterviewSourceRow source={s} change={null} readOnly={false} />
      </ul>
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.patch.mockReset().mockResolvedValue({ data: source({ body: "New notes" }) });
});

describe("InterviewSourceRow", () => {
  it("edits pasted text in place and saves its label and text", async () => {
    const user = userEvent.setup();
    renderRow(source({}));
    await user.click(screen.getByRole("button", { name: "Edit Huly" }));
    const text = screen.getByRole("textbox", { name: "Text" });
    expect(text).toHaveValue("Old notes");
    await user.clear(text);
    await user.type(text, "New notes");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(mocks.patch).toHaveBeenCalledWith("/api/memories/interview-sources/s-1", { label: "Huly", body: "New notes" });
    expect(await screen.findByRole("button", { name: "Edit Huly" })).toBeInTheDocument();
  });

  it("refuses emptied text without saving", async () => {
    const user = userEvent.setup();
    renderRow(source({}));
    await user.click(screen.getByRole("button", { name: "Edit Huly" }));
    await user.clear(screen.getByRole("textbox", { name: "Text" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Paste some text")).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("offers no edit on a source that lives elsewhere", () => {
    renderRow(source({ kind: "doc", ref: "d-1", body: "" }));
    expect(screen.queryByRole("button", { name: /^Edit/ })).not.toBeInTheDocument();
  });
});
