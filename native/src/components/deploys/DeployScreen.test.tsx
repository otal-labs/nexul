import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { DeployScreen } from "@/components/deploys/DeployScreen";
import { dispatch } from "@/hooks/useLiveEvents";
import type { DeployLogLine } from "@/models/Stack";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

jest.mock("expo-router", () => ({ useLocalSearchParams: () => ({ id: "d-1" }), Stack: { Screen: () => null } }));

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

const deploy = { id: "d-1", stack_id: "s-1", image: "api:2", status: "running", created_at: "2026-09-29T10:00:00Z" };
let lines: DeployLogLine[] = [];

const mockGet = (url: string) => {
  if (url === "/api/deploys/d-1") return Promise.resolve(deploy);
  if (url === "/api/deploys/d-1/log") return Promise.resolve(lines);
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
    expect(screen.getByText("running")).toBeTruthy();
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

    expect(await screen.findByText("Waiting for the runner to pick this up…")).toBeTruthy();
  });
});
