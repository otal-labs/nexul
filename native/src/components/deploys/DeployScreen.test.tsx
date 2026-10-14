import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { DeployScreen } from "@/components/deploys/DeployScreen";
import { dispatch } from "@/hooks/useLiveEvents";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { DeployLogLine } from "@/models/Stack";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useLocalSearchParams: () => ({ id: "d-1" }), useRouter: () => ({ push: mockPush }), Stack: { Screen: () => null } }));

// dispatch pulls in useLiveEvents' sessionStore import chain, which wires the query client's online manager
// to expo-network/expo-secure-store; both are stubbed the same way useLiveEvents.test.tsx does it.
jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));
jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));

// Jest has no layout pass, so this stand-in renders every row in data order and keeps the props for the
// follow-the-tail assertion, same approach as ChatThreadScreen's MessageList mock.
const mockListProps: { current: Record<string, unknown> } = { current: {} };
jest.mock("@legendapp/list/react-native", () => ({
  LegendList: (props: {
    data: unknown[];
    keyExtractor: (item: unknown) => string;
    renderItem: (info: { item: unknown }) => unknown;
  }) => {
    const { Fragment, createElement } = jest.requireActual<typeof import("react")>("react");
    mockListProps.current = props;
    return props.data.map((item) => createElement(Fragment, { key: props.keyExtractor(item) }, props.renderItem({ item }) as never));
  },
}));

notifyManager.setScheduler((callback) => callback());

let deploy = { id: "d-1", stack_id: "s-1", image: "api:2", status: "running", created_at: "2026-09-29T10:00:00Z" };
let lines: DeployLogLine[] = [];
let me: MyWorkspaceInfo;

const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve(me);
  if (url === "/api/deploys/d-1") return Promise.resolve(deploy);
  if (url === "/api/deploys/d-1/log") return Promise.resolve(lines);
  if (url === "/api/stacks/s-1") return Promise.resolve({ id: "s-1", project_id: "p-1", name: "api" });
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  await render(
    <QueryClientProvider client={client}>
      <DeployScreen />
    </QueryClientProvider>,
  );
  return client;
};

beforeEach(() => {
  mockPush.mockReset();
  me = { role_name: "Developer", permissions: ["deploys:write"], show_locked_actions: false };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  deploy = { id: "d-1", stack_id: "s-1", image: "api:2", status: "running", created_at: "2026-09-29T10:00:00Z" };
  jest.mocked(api.get).mockReset().mockImplementation(mockGet);
  lines = [
    { seq: 1, ts: 1_000, phase: "deploy", text: "starting" },
    { seq: 2, ts: 2_000, phase: "deploy", text: "pulling image" },
  ];
});

describe("DeployScreen", () => {
  test("renders the deploy's status and every log line in order", async () => {
    await renderScreen();

    expect(await screen.findByText("starting")).toBeTruthy();
    expect(screen.getByText("pulling image")).toBeTruthy();
    expect(screen.getByText("Deploying")).toBeTruthy();
    expect(mockListProps.current).toMatchObject({ initialScrollAtEnd: true, alignItemsAtEnd: true, maintainScrollAtEnd: true });
  });

  test("a live deploy.updated event appends the new line and keeps following the tail", async () => {
    const client = await renderScreen();
    await screen.findByText("pulling image");

    lines = [...lines, { seq: 3, ts: 3_000, phase: "deploy", text: "container healthy" }];
    const invalidate = jest.spyOn(client, "invalidateQueries");
    await act(async () => {
      dispatch(client)({ topic: "deploy.updated", type: "event", payload: {} });
      await Promise.all(invalidate.mock.results.map((result) => result.value));
    });

    expect(await screen.findByText("container healthy")).toBeTruthy();
    expect(mockListProps.current).toMatchObject({ maintainScrollAtEnd: true });
  });

  test("an empty log shows the waiting message instead of the list", async () => {
    lines = [];
    await renderScreen();

    expect(await screen.findByText("Waiting for the runner to pick this up")).toBeTruthy();
  });

  test("a running deploy offers Cancel deploy to someone who can deploy the stack, and opens the confirm", async () => {
    await renderScreen();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Cancel deploy" }));

    expect(mockPush).toHaveBeenCalledWith({ pathname: "/deploys/cancel", params: { deployId: "d-1" } });
  });

  test("Cancel deploy is left out once the deploy has finished", async () => {
    deploy = { ...deploy, status: "healthy" };
    await renderScreen();

    await screen.findByText("starting");
    expect(screen.queryByRole("button", { name: "Cancel deploy" })).toBeNull();
  });

  test.each([
    ["is hidden from a role that can't deploy", { permissions: ["stacks:read"], show_locked_actions: false }, undefined],
    ["is locked for a role that shows locked actions", { permissions: ["stacks:read"], show_locked_actions: true }, "Your role can't cancel deploys"],
  ])("Cancel deploy %s", async (_name, role, hint) => {
    me = { role_name: "Client", ...role };
    await renderScreen();

    await screen.findByText("starting");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await act(async () => {});
    const control = screen.queryByRole("button", { name: "Cancel deploy" });
    expect(control === null).toBe(hint === undefined);
    expect(control?.props.accessibilityHint).toBe(hint);
  });
});
