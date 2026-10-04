import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InterviewChecklistFeed } from "@/components/memory/InterviewChecklistFeed";
import { buildSections, type InterviewAnswer } from "@/models/InterviewAnswer";
import type { InterviewDraft, InterviewSource } from "@/models/InterviewSource";
import type { InterviewQuestion } from "@/models/InterviewTemplate";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), post: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: (e: Error) => e.message }));

const options = (...labels: string[]) => labels.map((label) => ({ label, description: "" }));
const questions: InterviewQuestion[] = [
  { text: "When are tests written?", hint: "", multi_select: false, options: options("Before the code", "With the change") },
  { text: "How is the code organised?", hint: "", multi_select: false, options: options("Layers", "Feature folders") },
];

const source: InterviewSource = {
  id: "s-1", workspace_id: "ws-1", project_id: "p-1", kind: "doc", ref: "d-1", label: "Engineering standards",
  stance: "follow", added_by: "u-1", added_at: "2026-10-01T10:00:00Z", updated_at: "2026-10-01T10:00:00Z",
};

const draft = (id: string, question: string, selected: string[], at: string): InterviewDraft => ({
  id, workspace_id: "ws-1", project_id: "p-1", question, selected, text: "", source_ids: ["s-1"],
  where: '"A change lands with its tests"', drafted_by: "u-1", drafted_at: at,
});

const answer = (question: string, selected: string[], at: string): InterviewAnswer => ({
  id: `a-${question}`, workspace_id: "ws-1", project_id: "p-1", round: 0, question, selected, text: "",
  skipped: false, answered_by: "u-1", answered_at: at,
});

const renderFeed = (answers: InterviewAnswer[], drafts: InterviewDraft[], asked = questions) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <InterviewChecklistFeed projectId="p-1" sections={buildSections(asked, answers, drafts, [source])} readOnly={false} hasMemory memoryWithoutAnswers={false} />
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.get.mockReset().mockResolvedValue({ data: [] });
  mocks.put.mockReset().mockImplementation((_url: string, body: object) => Promise.resolve({ data: { ...body, id: "a-new", answered_at: "" } }));
  mocks.delete.mockReset().mockResolvedValue({ data: undefined });
});

describe("InterviewChecklistFeed with drafts", () => {
  it("opens a drafted question on the draft, with where it came from, and Next confirms it", async () => {
    const user = userEvent.setup();
    renderFeed([], [draft("d-1", "When are tests written?", ["With the change"], "2026-10-02T10:00:00Z")]);
    expect(screen.getByText(/From/)).toHaveTextContent('From Engineering standards: "A change lands with its tests"');
    expect(screen.getByRole("radio", { name: "With the change" })).toBeChecked();
    expect(screen.getByText("Drafted", { selector: "span.uppercase" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/memories/interview-answers", {
      project_id: "p-1", round: 0, question: "When are tests written?", selected: ["With the change"], text: "",
    });
  });

  it("shows a newer, different draft on an answer as a suggested change and Accept saves it", async () => {
    const user = userEvent.setup();
    renderFeed(
      [answer("When are tests written?", ["With the change"], "2026-10-02T10:00:00Z"), answer("How is the code organised?", ["Layers"], "2026-10-02T10:00:00Z")],
      [draft("d-2", "When are tests written?", ["Before the code"], "2026-10-03T10:00:00Z")],
    );
    expect(screen.getByText("2 of 2 answered · 1 suggested")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /When are tests written/ }));
    expect(screen.getByText("Your answer").nextSibling).toHaveTextContent("With the change");
    await user.click(screen.getByRole("button", { name: "Accept" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/memories/interview-answers", {
      project_id: "p-1", round: 0, question: "When are tests written?", selected: ["Before the code"], text: "",
    });
  });

  it("dismisses a suggested change through the draft route", async () => {
    const user = userEvent.setup();
    renderFeed(
      [answer("When are tests written?", ["With the change"], "2026-10-02T10:00:00Z")],
      [draft("d-3", "When are tests written?", ["Before the code"], "2026-10-03T10:00:00Z")],
    );
    await user.click(screen.getByRole("button", { name: /When are tests written/ }));
    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/memories/interview-drafts/d-3");
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("gives a question without options a text box where Enter is a new line and Next saves", async () => {
    const user = userEvent.setup();
    const stack = "What languages and frameworks does this project use?";
    renderFeed([], [{ ...draft("d-5", stack, [], "2026-10-02T10:00:00Z"), text: "Go 1.24" }], [{ text: stack, hint: "", multi_select: false, options: [] }]);
    const box = screen.getByRole("textbox", { name: "Your answer" });
    expect(box.tagName).toBe("TEXTAREA");
    expect(box).toHaveValue("Go 1.24");
    await user.type(box, "{Enter}React 19");
    expect(mocks.put).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/memories/interview-answers", {
      project_id: "p-1", round: 0, question: stack, selected: [], text: "Go 1.24\nReact 19",
    });
  });

  it("leaves an answer the person changed from an older draft alone", () => {
    renderFeed(
      [answer("When are tests written?", ["Before the code"], "2026-10-03T10:00:00Z")],
      [draft("d-4", "When are tests written?", ["With the change"], "2026-10-02T10:00:00Z")],
    );
    expect(screen.queryByText("Suggested change")).not.toBeInTheDocument();
  });
});
