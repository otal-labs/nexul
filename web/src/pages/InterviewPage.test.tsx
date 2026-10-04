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
import type { Trail } from "@/models/Trail";
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

const renderPage = (entry = "/acme/projects/BE/interview", client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) =>
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/acme/projects/:projectId/interview" element={<InterviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

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
  trails?: () => Trail[];
}

const mockApi = ({ memories = [], questions = [organised, testing], answers = [], plays = [], trails = () => [] }: Setup = {}) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/plays/runs") return { data: trails() };
    if (url === "/api/memories") return { data: memories };
    if (url === "/api/memories/interview-template") return { data: { questions } };
    if (url === "/api/memories/interview-answers") return { data: answers };
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: plays };
    return { data: [] };
  });

const waitingTrail = (patch: Partial<Trail> = {}): Trail => ({
  id: "tr-1", workspace_id: "ws-1", play_id: "play-interview", play_label: "Interview", target_type: "interview", target_id: "p-1",
  project_id: "p-1", conversation_id: "c-1", starter_id: "u-1", via: "web", selected_memory_ids: [], custom_instructions: "",
  computer_id: "", provider: "", model: "", harness_session_id: "", state: "waiting", started_at: "", ended_at: null, last_error: "",
  failure_reason: "", reply_message_id: "", activity: [],
  question: {
    request_id: "req-1",
    asked_at: "",
    questions: [
      {
        id: "runner",
        text: "Which test runner? ci.yml runs both bun test and vitest.",
        header: "Tests",
        options: [{ label: "Vitest (Recommended)", value: "vitest" }, { label: "Bun" }],
      },
      { id: "floor", text: "What coverage floor?", header: "Coverage", options: [] },
    ],
  },
  ...patch,
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

  it("never offers the drafting play as the interview's run", async () => {
    const draftPlay: Play = { ...interviewPlay, id: "play-draft", label: "Draft interview", description: "Drafts answers.", builtin_key: "interview-draft" };
    mockApi({ answers: [stored(0, organised.text), stored(0, testing.text, { skipped: true, text: "" })], plays: [draftPlay, interviewPlay] });
    renderPage();
    expect(await screen.findByRole("button", { name: /Done/ })).toHaveAttribute("title", interviewPlay.description);
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

  it("marks the memory out of date once an answer changes, and clears it once a run regenerates it", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    let trails = [waitingTrail({ id: "tr-0", state: "done", question: null, ended_at: "2026-10-03T10:00:00Z" })];
    let answers = [stored(0, organised.text, { answered_at: "2026-10-03T09:00:00Z" }), stored(0, testing.text, { answered_at: "2026-10-03T09:00:00Z" })];
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [project] };
      if (url === "/api/plays/runs") return { data: trails };
      if (url === "/api/memories") return { data: [{ ...interview, updated_at: "2026-10-03T10:00:00Z" }] };
      if (url === "/api/memories/interview-template") return { data: { questions: [organised, testing] } };
      if (url === "/api/memories/interview-answers") return { data: answers };
      if (url === "/api/workspaces/ws-1/plays/applicable") return { data: [interviewPlay] };
      return { data: [] };
    });
    vi.mocked(api.put).mockImplementation(async () => {
      answers = [stored(0, organised.text, { text: "", selected: ["Feature folders"], answered_at: "2026-10-03T11:00:00Z" }), answers[1]!];
      return { data: answers[0] };
    });
    const user = userEvent.setup();
    renderPage(undefined, client);

    expect(await screen.findByRole("button", { name: /^Regenerate$/ })).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "out of date" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /How is the code organised\?/ }));
    await user.click(screen.getByRole("radio", { name: "Feature folders" }));
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(await screen.findByRole("button", { name: /Regenerate the memory/ })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "out of date" })).toBeInTheDocument();
    expect(screen.getByText("· 1 answer changed")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Done" })).not.toBeInTheDocument();

    trails = [waitingTrail({ id: "tr-1", state: "done", question: null, ended_at: "2026-10-03T11:05:00Z" }), ...trails];
    await client.invalidateQueries();

    expect(await screen.findByRole("button", { name: /^Regenerate$/ })).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "out of date" })).not.toBeInTheDocument();
  });

  it("keeps the memory out of date after a hand edit, since only a run reads the changed answer", async () => {
    mockApi({
      memories: [{ ...interview, updated_at: "2026-10-03T12:00:00Z" }],
      answers: [stored(0, organised.text, { answered_at: "2026-10-03T11:00:00Z" })],
      plays: [interviewPlay],
      trails: () => [waitingTrail({ state: "done", question: null, ended_at: "2026-10-03T10:00:00Z" })],
    });
    renderPage();
    expect(await screen.findByRole("button", { name: /Regenerate the memory/ })).toBeInTheDocument();
  });

  it("offers Regenerate the memory for a memory from before its questions once one is answered", async () => {
    mockApi({ memories: [interview], answers: [stored(0, organised.text, { answered_at: "2026-10-03T11:00:00Z" })], plays: [interviewPlay] });
    renderPage();
    expect(await screen.findByRole("button", { name: /Regenerate the memory/ })).toBeInTheDocument();
    expect(screen.getByText("· 1 answer changed")).toBeInTheDocument();
  });

  it("offers Done again after a run that failed before writing the memory", async () => {
    mockApi({
      answers: [stored(0, organised.text), stored(0, testing.text), stored(1, "Where do fixtures live?", { text: "test/" })],
      plays: [interviewPlay],
      trails: () => [waitingTrail({ state: "failed", question: null, ended_at: "2026-10-03T10:00:00Z" })],
    });
    renderPage();
    expect(await screen.findByText("Run failed")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Done" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Regenerate/ })).not.toBeInTheDocument();
  });

  it("shows a waiting run's question as the next round, opened with its recommended option picked", async () => {
    mockApi({ answers: [stored(0, organised.text), stored(0, testing.text)], trails: () => [waitingTrail()] });
    renderPage();

    expect(await section(/Which test runner\?/)).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("button", { name: /Follow-ups from the agent 1/ })).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("button", { name: /Initial questions/ })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Follow-up 1 of 2")).toBeInTheDocument();
    expect(screen.getByText("ci.yml runs both bun test and vitest.")).toBeInTheDocument();
    expect(screen.queryByText("Tests")).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Vitest (Recommended)" })).toBeChecked();
    expect(screen.queryByRole("button", { name: /Done/ })).not.toBeInTheDocument();
  });

  it("sends the whole round on the trail, then shows the stored round in its place", async () => {
    let trail = waitingTrail();
    let answers = [stored(0, organised.text), stored(0, testing.text)];
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [project] };
      if (url === "/api/plays/runs") return { data: [trail] };
      if (url === "/api/memories/interview-template") return { data: { questions: [organised, testing] } };
      if (url === "/api/memories/interview-answers") return { data: answers };
      return { data: [] };
    });
    vi.mocked(api.post).mockImplementation(async () => {
      trail = waitingTrail({ state: "running", question: null });
      answers = [
        ...answers,
        stored(1, "Which test runner?", { why: "ci.yml runs both bun test and vitest.", selected: ["Vitest (Recommended)"], text: "" }),
        stored(1, "What coverage floor?", { text: "85" }),
      ];
      return { data: trail };
    });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Next" }));
    expect(api.post).not.toHaveBeenCalled();
    expect(await section(/What coverage floor\?/)).toHaveAttribute("aria-expanded", "true");
    await user.type(screen.getByRole("textbox", { name: "Your answer" }), "85");
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/answer", {
      answers: { runner: { selected: ["vitest"] }, floor: { text: "85" } },
    });
    expect(await screen.findByText("85")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Follow-ups from the agent 1/ })).toHaveTextContent("2 of 2 answered");
    expect(screen.queryByRole("radio", { name: "Vitest (Recommended)" })).not.toBeInTheDocument();
  });

  it("sends a skipped live follow-up as Skipped, which the harness accepts where an empty answer may not be", async () => {
    mockApi({ answers: [stored(0, organised.text), stored(0, testing.text)], trails: () => [waitingTrail()] });
    vi.mocked(api.post).mockResolvedValue({ data: waitingTrail({ state: "running", question: null }) });
    const user = userEvent.setup();
    renderPage();

    await section(/Which test runner\?/);
    await user.click(screen.getByRole("button", { name: "Skip" }));
    expect(await section(/Which test runner\?/)).toHaveTextContent("Skipped");
    await user.type(screen.getByRole("textbox", { name: "Your answer" }), "85");
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/answer", {
      answers: { runner: { text: "Skipped" }, floor: { text: "85" } },
    });
  });
});
