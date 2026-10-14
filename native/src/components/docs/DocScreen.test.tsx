import type { Clarification, ClarificationQuestion, ClarificationRound } from "@nexul/client-core/clarification";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { DocScreen } from "@/components/docs/DocScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), put: jest.fn(), delete: jest.fn() } }));
jest.mock("expo-router", () => ({ Stack: { Screen: () => null } }));
jest.mock("@/components/docs/DocBody", () => ({
  DocBody: ({ body }: { body: string }) => jest.requireActual("react").createElement(jest.requireActual("react-native").Text, null, body),
}));
jest.mock("@/hooks/ProjectHooks", () => ({ useRevokedProject: () => false }));

let mockCanWrite = true;
jest.mock("@/hooks/WorkspaceHooks", () => ({ useAreaAccess: () => (area: string) => area === "editDocs" && mockCanWrite }));

const doc = {
  id: "doc-1",
  project_id: "proj-1",
  title: "Booking app",
  body: "The doc body",
  version: 3,
  archived: false,
  locked: false,
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-02T00:00:00Z",
};

const question = (id: string, extra: Partial<ClarificationQuestion> = {}): ClarificationQuestion => ({
  id, doc_id: "doc-1", round: 1, position: 1, question: `Question ${id}?`, why: "The doc leaves it open.",
  options: [{ label: "By service (Suggested)" }, { label: "One fixed slot" }], multi_select: false,
  selected: [], text: "", skipped: false, ...extra,
});

const round = (questions: ClarificationQuestion[], extra: Partial<ClarificationRound> = {}): ClarificationRound => ({
  doc_id: "doc-1", round: 1, started_by: "u1", trail_id: "t1", started_at: "2026-10-02T00:00:00Z", running: false,
  anything_else: "", anything_else_reply: "", questions, ...extra,
});

const clarification = (rounds: ClarificationRound[], extra: Partial<Clarification> = {}): Clarification => ({
  rounds, running: false, closed: false, can_answer: true, can_close: false, ...extra,
});

const waiting = clarification([round([question("q1"), question("q2")])]);

const serve = (clarificationBody: Clarification | ApiError) =>
  jest.mocked(api.get).mockImplementation((path: string) => {
    if (path === "/api/docs/doc-1") return Promise.resolve(doc);
    if (path !== "/api/docs/doc-1/clarification") return Promise.reject(new Error(`unexpected GET ${path}`));
    if (clarificationBody instanceof ApiError) return Promise.reject(clarificationBody);
    return Promise.resolve(clarificationBody);
  });

const renderScreen = async (openedOn?: "doc" | "questions") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const rendered = await render(
    <QueryClientProvider client={client}>
      <DocScreen docId="doc-1" {...(openedOn && { openedOn })} />
    </QueryClientProvider>,
  );
  return { ...rendered, client };
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.put).mockReset();
  jest.mocked(api.delete).mockReset();
  mockCanWrite = true;
});

describe("DocScreen questions", () => {
  test("opens on the doc, with the waiting count on the Questions side as its only marker", async () => {
    serve(waiting);
    await renderScreen();

    expect(await screen.findByText("The doc body")).toBeTruthy();
    expect(await screen.findByLabelText("Questions, 2 waiting")).toBeTruthy();
    expect(screen.queryByText("2 questions waiting")).toBeNull();
  });

  test("opened on Questions it goes straight to them, never showing the doc first", async () => {
    serve(waiting);
    await renderScreen("questions");

    expect(await screen.findByText("2 questions waiting")).toBeTruthy();
    expect(screen.queryByText("The doc body")).toBeNull();
  });

  test("a doc with no rounds, or on a server that has no questions, reads as a plain doc", async () => {
    serve(clarification([]));
    const { unmount } = await renderScreen("questions");
    expect(await screen.findByText("The doc body")).toBeTruthy();
    expect(screen.queryByLabelText(/^Questions/)).toBeNull();
    await unmount();

    serve(new ApiError(404, null, "not found"));
    await renderScreen("questions");
    expect(await screen.findByText("The doc body")).toBeTruthy();
    expect(screen.queryByLabelText(/^Questions/)).toBeNull();
  });

  test("the open question starts with nothing picked, the suggested option included, and Next saves the pick", async () => {
    serve(waiting);
    jest.mocked(api.put).mockResolvedValue(question("q1", { selected: ["One fixed slot"] }));
    await renderScreen("questions");
    const user = userEvent.setup();

    expect(await screen.findByText("Question 1 of 2")).toBeTruthy();
    expect(screen.getByRole("radio", { name: "By service (Suggested)" })).not.toBeChecked();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();

    await user.press(screen.getByRole("radio", { name: "One fixed slot" }));
    await user.press(screen.getByRole("button", { name: "Next" }));

    expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/questions/q1", { selected: ["One fixed slot"], text: "", skipped: false });
  });

  test("Skip saves the question as skipped", async () => {
    serve(waiting);
    jest.mocked(api.put).mockResolvedValue(question("q1", { skipped: true }));
    await renderScreen("questions");

    await userEvent.setup().press(await screen.findByRole("button", { name: "Skip" }));

    expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/questions/q1", { selected: [], text: "", skipped: true });
  });

  test("an answered row shows its whole answer, however long, never cut to one line", async () => {
    const long = "Charged per service, and bigger dogs take longer, so the groomer adds time at the desk when they book";
    serve(clarification([round([question("q1", { selected: [long]}), question("q2")])]));
    await renderScreen("questions");

    const answer = await screen.findByText(long);

    expect(answer.props.numberOfLines).toBeUndefined();
  });

  test("a reader sees the questions and answers and no way to change them, whatever their role holds", async () => {
    serve(clarification([round([question("q1", { selected: ["One fixed slot"] }), question("q2")])], { can_answer: false }));
    await renderScreen("questions");

    expect(await screen.findByText("One fixed slot")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Next" })).toBeNull();
    expect(screen.queryByLabelText("Anything else?")).toBeNull();
  });

  test("a member who may edit only this doc, through a rule on it, answers though their role cannot", async () => {
    mockCanWrite = false;
    serve(waiting);
    await renderScreen("questions");

    expect(await screen.findByRole("button", { name: "Next" })).toBeTruthy();
    expect(screen.getByLabelText("Anything else?")).toBeTruthy();
  });

  test("an answer someone else saves to the open question shows in its row and opens the next one", async () => {
    serve(waiting);
    const { client } = await renderScreen("questions");
    expect(await screen.findByText("Question 1 of 2")).toBeTruthy();

    serve(clarification([round([question("q1", { selected: ["By service (Suggested)"] }), question("q2")])]));
    await act(() => client.invalidateQueries());

    expect(await screen.findByText("Question 2 of 2")).toBeTruthy();
    expect(screen.queryByText("Question 1 of 2")).toBeNull();
    expect(screen.getByText("By service")).toBeTruthy();
  });

  test("an answer someone else saves to the open question leaves what this person has typed in it", async () => {
    serve(waiting);
    const { client } = await renderScreen("questions");
    await userEvent.setup().press(await screen.findByRole("radio", { name: "One fixed slot" }));

    serve(clarification([round([question("q1", { selected: ["By service (Suggested)"] }), question("q2")])]));
    await act(() => client.invalidateQueries());

    expect(await screen.findByText("Question 1 of 2")).toBeTruthy();
    expect(screen.getByRole("radio", { name: "One fixed slot" })).toBeChecked();
  });

  test("Anything else? saves the text of the newest round", async () => {
    serve(waiting);
    jest.mocked(api.put).mockResolvedValue(round([], { anything_else: "Can people pick a groomer?" }));
    await renderScreen("questions");

    await fireEvent.changeText(await screen.findByLabelText("Anything else?"), "Can people pick a groomer?");
    await userEvent.setup().press(screen.getByRole("button", { name: "Save" }));

    expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/rounds/1/anything-else", { text: "Can people pick a groomer?" });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/docs/doc-1/clarification"));
  });

  test("while a round runs, an answerer sees only that more questions are coming and a closer sees the run", async () => {
    serve(clarification([round([], { running: true })], { running: true }));
    const { unmount } = await renderScreen("questions");
    expect(await screen.findByText("More questions coming")).toBeTruthy();
    await unmount();

    serve(clarification([round([], { running: true })], { running: true, can_close: true }));
    await renderScreen("questions");
    expect(await screen.findByText("Round 1 running")).toBeTruthy();
  });
});
