import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import type { InterviewAnswer } from "@/models/InterviewAnswer";
import type { InterviewQuestion } from "@/models/InterviewTemplate";
import type { Memory } from "@/models/Memory";
import type { Play } from "@/models/Play";
import { InterviewPage } from "@/pages/InterviewPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
const perms = vi.hoisted(() => ({ denied: [] as string[] }));
vi.mock("@/hooks/WorkspaceHooks", () => ({ useHasPermission: (value: string) => !perms.denied.includes(value) }));
const access = vi.hoisted(() => ({ areas: ["tickets"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));
vi.mock("@/components/doc/DocBodyView", () => ({ DocBodyView: ({ body }: { body: string }) => <div>{body}</div> }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const interview: Memory = {
  id: "mem-i",
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "interview",
  title: "Interview",
  when_to_use: "",
  body: "## Stack",
  always_included: true,
  footer: false,
  version: 1,
  created_by: "u-1",
  created_at: "",
  updated_by: "u-1",
  updated_at: "",
};

const renderPage = (entry = "/acme/projects/BE/interview") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/acme/projects/:projectId/interview" element={<InterviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const interviewPlay: Play = {
  id: "play-interview",
  workspace_id: "ws-1",
  label: "Interview",
  type: "interview",
  description: "Asks one question at a time.",
  instructions: "",
  enabled: true,
  show_when_stage: null,
  excluded_project_ids: [],
  builtin_key: "",
  created_by: "",
  created_at: "",
  updated_at: "",
};

const organised: InterviewQuestion = {
  text: "How is the code organised?",
  hint: "",
  multi_select: false,
  options: [
    { label: "Layers", description: "" },
    { label: "Feature folders", description: "" },
  ],
};
const testing: InterviewQuestion = { text: "When are tests written?", hint: "", multi_select: false, options: [] };
const vocabulary: InterviewQuestion = { text: "Which words mean something here?", hint: "", multi_select: false, options: [] };

const stored = (round: number, question: string, patch: Partial<InterviewAnswer> = {}): InterviewAnswer => ({
  id: `a-${round}-${question}`,
  workspace_id: "ws-1",
  project_id: "p-1",
  round,
  question,
  selected: [],
  text: "With the change",
  skipped: false,
  answered_by: "u-1",
  answered_at: "",
  ...patch,
});

interface Setup {
  memories?: Memory[];
  questions?: InterviewQuestion[];
  answers?: InterviewAnswer[];
  plays?: Play[];
}

const mockApi = ({ memories = [], questions = [organised, testing], answers = [], plays = [] }: Setup = {}) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/memories") return { data: memories };
    if (url === "/api/memories/interview-template") return { data: { questions } };
    if (url === "/api/memories/interview-answers") return { data: answers };
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: plays };
    return { data: [] };
  });

const section = (name: RegExp) => screen.findByRole("button", { name });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.put).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  access.areas = ["tickets"];
  perms.denied = [];
});

describe("InterviewPage", () => {
  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect((await screen.findAllByText("Something went wrong")).length).toBeGreaterThan(0);
  });

  it("says so when the project does not exist", async () => {
    mockApi();
    renderPage("/acme/projects/NOPE/interview");
    expect(await screen.findByText("Project not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to your board" })).toHaveAttribute("href", "/acme/board");
  });

  it("does not send a viewer who can't read tickets to the board from a missing project", async () => {
    access.areas = [];
    mockApi();
    renderPage("/acme/projects/NOPE/interview");
    expect(await screen.findByText("Project not found")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Go to your board" })).not.toBeInTheDocument();
  });

  it("opens the first question neither answered nor skipped and counts the rest", async () => {
    mockApi({
      questions: [organised, testing, vocabulary],
      answers: [stored(0, organised.text, { text: "", selected: ["Layers"] }), stored(0, testing.text, { skipped: true, text: "" })],
    });
    renderPage();

    expect(await section(/Which words mean something here\?/)).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("button", { name: /How is the code organised\?/ })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Layers")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Initial questions/ })).toHaveTextContent("1 answered · 1 skipped");
    expect(screen.getByText("No memory yet")).toBeInTheDocument();
  });

  it("saves the answer on Next and opens the next question", async () => {
    mockApi();
    vi.mocked(api.put).mockResolvedValue({ data: stored(0, organised.text, { text: "", selected: ["Layers"] }) });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("radio", { name: "Layers" }));
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(api.put).toHaveBeenCalledWith("/api/memories/interview-answers", {
      project_id: "p-1",
      round: 0,
      question: organised.text,
      selected: ["Layers"],
      text: "",
    });
    expect(await section(/When are tests written\?/)).toHaveAttribute("aria-expanded", "true");
  });

  it("saves a skip through the skip route", async () => {
    mockApi();
    vi.mocked(api.post).mockResolvedValue({ data: stored(0, organised.text, { skipped: true, text: "" }) });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Skip" }));

    expect(api.post).toHaveBeenCalledWith("/api/memories/interview-answers/skip", {
      project_id: "p-1",
      round: 0,
      question: organised.text,
      selected: [],
      text: "",
    });
  });

  it("folds the initial questions once the agent has asked a round of follow-ups", async () => {
    mockApi({
      answers: [
        stored(0, organised.text),
        stored(0, testing.text),
        stored(1, "Where do fixtures live?", { why: "The checkout has two folders.", text: "" }),
      ],
    });
    renderPage();

    expect(await section(/Initial questions/)).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("button", { name: /Follow-ups from the agent 1/ })).toHaveTextContent("0 of 1 answered");
    expect(screen.getByText("The checkout has two folders.")).toBeInTheDocument();
  });

  it("offers Done to start the run once every initial question is answered or skipped", async () => {
    mockApi({ answers: [stored(0, organised.text), stored(0, testing.text, { skipped: true, text: "" })], plays: [interviewPlay] });
    renderPage();
    expect(await screen.findByRole("button", { name: /Done/ })).toBeInTheDocument();
  });

  it("shows the answers read-only to someone who may not answer", async () => {
    perms.denied = ["memories:write"];
    mockApi({ answers: [stored(0, organised.text)] });
    renderPage();

    expect(await screen.findByText("With the change")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /How is the code organised\?/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
  });

  it("shows an existing memory beside unanswered questions, with Regenerate", async () => {
    mockApi({ memories: [interview], plays: [interviewPlay] });
    renderPage();

    expect(await screen.findByText("This memory came from an earlier interview. Answering these questions lets the agent update it.")).toBeInTheDocument();
    expect(screen.getByText("## Stack")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Initial questions/ })).toHaveTextContent("0 of 2 answered");
    expect(await screen.findByRole("button", { name: /Regenerate/ })).toBeInTheDocument();
  });
});
