import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InterviewDraftRunLine } from "@/components/memory/InterviewDraftRunLine";
import type { InterviewAnswer } from "@/models/InterviewAnswer";
import type { InterviewDraft } from "@/models/InterviewSource";
import type { InterviewQuestion } from "@/models/InterviewTemplate";
import type { Trail } from "@/models/Trail";

const mocks = vi.hoisted(() => ({ drafts: vi.fn(), answers: vi.fn() }));

const RUN_START = "2026-10-04T10:00:00Z";
const run = { id: "t-2", state: "done", started_at: RUN_START } as Trail;

vi.mock("@/hooks/InterviewSourceHooks", () => ({
  useInterviewTrails: () => ({ drafting: [run] }),
  useFetchInterviewDrafts: () => ({ data: mocks.drafts() }),
}));

const questions: InterviewQuestion[] = ["Stack?", "Tests?", "Style?"].map((text) => ({ text, hint: "", multi_select: false, options: [] }));

vi.mock("@/hooks/MemoryHooks", () => ({
  useFetchProjectInterviewTemplate: () => ({ data: { questions } }),
  useFetchInterviewAnswers: () => ({ data: mocks.answers() }),
}));

const draft = (question: string, text: string, at = "2026-10-04T10:05:00Z"): InterviewDraft => ({
  id: `d-${question}`, workspace_id: "ws-1", project_id: "p-1", question, selected: [], text, source_ids: ["s-1"], where: "",
  drafted_by: "u-1", drafted_at: at,
});

const answer = (question: string, text: string, at = "2026-10-03T10:00:00Z"): InterviewAnswer => ({
  id: `a-${question}`, workspace_id: "ws-1", project_id: "p-1", round: 0, question, selected: [], text, skipped: false,
  answered_by: "u-1", answered_at: at,
});

beforeEach(() => {
  mocks.drafts.mockReset();
  mocks.answers.mockReset();
});

describe("InterviewDraftRunLine", () => {
  it("counts a redraft's changes to answered questions as suggested, not drafted", () => {
    mocks.answers.mockReturnValue([answer("Stack?", "Go"), answer("Tests?", "Mocks"), answer("Style?", "Ours")]);
    mocks.drafts.mockReturnValue([draft("Stack?", "Go and React"), draft("Tests?", "Real SQLite"), draft("Style?", "Early return")]);
    render(<InterviewDraftRunLine projectId="p-1" />);
    expect(screen.getByText(/· /)).toHaveTextContent("· 3 suggested");
  });

  it("stops counting a draft once it is confirmed, and leaves out drafts from an earlier run", () => {
    mocks.answers.mockReturnValue([answer("Stack?", "Go and React", "2026-10-04T10:10:00Z"), answer("Tests?", "Mocks")]);
    mocks.drafts.mockReturnValue([draft("Stack?", "Go and React"), draft("Tests?", "Real SQLite"), draft("Style?", "Ours", "2026-10-02T10:00:00Z")]);
    render(<InterviewDraftRunLine projectId="p-1" />);
    expect(screen.getByText(/· /)).toHaveTextContent("· 1 suggested");
  });

  it("counts drafts on unanswered questions out of the template", () => {
    mocks.answers.mockReturnValue([]);
    mocks.drafts.mockReturnValue([draft("Stack?", "Go"), draft("Tests?", "Real SQLite")]);
    render(<InterviewDraftRunLine projectId="p-1" />);
    expect(screen.getByText("Drafts ready")).toBeInTheDocument();
    expect(screen.getByText(/· /)).toHaveTextContent("· 2 of 3 drafted");
  });
});
