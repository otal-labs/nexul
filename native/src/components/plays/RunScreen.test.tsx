import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { RunScreen } from "@/components/plays/RunScreen";
import { dispatch } from "@/hooks/useLiveEvents";
import type { Trail } from "@/models/Trail";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));
jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: "r-1" }),
  Stack: { Screen: () => null },
}));
jest.mock("react-native-enriched-markdown", () => jest.requireActual("react-native-enriched-markdown/jest"));

notifyManager.setScheduler((callback) => callback());

const question = { request_id: "q", asked_at: "2026-10-01T10:02:00Z", questions: [{ id: "q1", text: "Ship it now?", options: [{ label: "Yes" }, { label: "Not yet" }] }] };

let trail: Trail;
let me: MyWorkspaceInfo;

const renderRun = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <RunScreen tab="board" />
    </QueryClientProvider>,
  );
  await screen.findByText("Fix with AI");
  return client;
};

beforeEach(() => {
  me = { role_name: "Developer", permissions: ["plays:read"], show_locked_actions: false };
  trail = {
    id: "r-1", workspace_id: "ws-1", play_id: "pl-1", play_label: "Fix with AI", target_type: "ticket", target_id: "t-1", project_id: "p-1",
    starter_id: "alice", custom_instructions: "", state: "waiting", started_at: "2026-10-01T10:00:00Z", ended_at: null, last_error: "",
    activity: [{ kind: "question", call_id: "c-q", tool: "AskUserQuestion", summary: "Ship it now?", at: "2026-10-01T10:02:00Z" }],
    question,
  };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.post).mockReset();
  jest.mocked(api.get).mockReset().mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { user: { id: "alice", login: "alice", name: "Alice" } };
    if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme" }];
    if (url === "/api/workspaces/ws-1/me") return me;
    if (url === "/api/plays/runs/r-1") return trail;
    throw new Error(`unexpected GET ${url}`);
  });
});

describe("RunScreen", () => {
  test("someone who neither started the run nor holds plays:write reads its question but can't answer or stop it", async () => {
    trail = { ...trail, starter_id: "bob" };
    await renderRun();

    expect(screen.getByText("Ship it now?", { exact: false })).toBeTruthy();
    expect(screen.getByText("Waiting for the starter's answer")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Send answer" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop run" })).toBeNull();
  });

  test("its starter answers the question in place, and the run goes on in the same trail", async () => {
    jest.mocked(api.post).mockResolvedValue({ ...trail, state: "running", question: { ...question, answer: { answers: { q1: { selected: ["Yes"] } } } } });
    await renderRun();
    const user = userEvent.setup();

    await user.press(await screen.findByRole("radio", { name: "Yes" }));
    await user.press(screen.getByRole("button", { name: "Send answer" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/plays/runs/r-1/answer", { answers: { q1: { selected: ["Yes"] } } }));
  });

  test("its starter stops it while it runs", async () => {
    trail = { ...trail, state: "running", question: null, activity: [] };
    jest.mocked(api.post).mockResolvedValue({ ...trail, state: "interrupted" });
    await renderRun();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Stop run" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/plays/runs/r-1/stop"));
  });

  test("a step frame lands in the open run without fetching it again", async () => {
    trail = { ...trail, state: "running", question: null, activity: [] };
    const client = await renderRun();
    const getsBefore = jest.mocked(api.get).mock.calls.length;

    await act(async () =>
      dispatch(client)({
        topic: "play.run",
        type: "direct",
        payload: {
          trail_id: "r-1", play_id: "pl-1", target_type: "ticket", target_id: "t-1", state: "running", question: null, ended_at: null, last_error: "",
          activity: { kind: "tool_call", call_id: "c-1", tool: "Bash", summary: "go test ./...", at: "2026-10-01T10:01:00Z" },
          starter_id: "alice", project_id: "p-1", workspace_id: "ws-1",
        },
      }),
    );

    expect(await screen.findAllByText("go test ./...", { exact: false })).not.toHaveLength(0);
    expect(jest.mocked(api.get).mock.calls.slice(getsBefore).map(([url]) => url)).not.toContain("/api/plays/runs/r-1");
  });
});
