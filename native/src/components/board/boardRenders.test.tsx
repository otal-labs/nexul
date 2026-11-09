import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { BoardScreen } from "@/components/board/BoardScreen";
import { useTicketPersonLookup } from "@/hooks/PeopleHooks";
import { getTicketsByProjectKey } from "@/hooks/TicketHooks";
import { useBoardStore } from "@/stores/boardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), patch: jest.fn(), post: jest.fn() } }));

const mockRouter = { push: jest.fn() };
jest.mock("expo-router", () => ({ useRouter: () => mockRouter }));

jest.mock("expo-sqlite/kv-store", () => ({
  __esModule: true,
  default: { getItemSync: () => null, setItemSync: jest.fn(), removeItemSync: jest.fn() },
}));

// Every card body resolves its person once per render, so the lookup's call count is the cards' render count.
jest.mock("@/hooks/PeopleHooks", () => ({
  ...jest.requireActual<typeof import("@/hooks/PeopleHooks")>("@/hooks/PeopleHooks"),
  useTicketPersonLookup: jest.fn(() => ({ login }: { login: string }) => ({ user_id: login, login, display_name: login, avatar_url: "" })),
}));

// Jest has no layout pass, so this stand-in renders the header and every row in data order.
jest.mock("@legendapp/list/react-native", () => ({
  LegendList: (props: {
    data: unknown[];
    keyExtractor: (item: unknown) => string;
    renderItem: (info: { item: unknown }) => unknown;
    ListHeaderComponent?: unknown;
  }) => {
    const { Fragment, createElement } = jest.requireActual<typeof import("react")>("react");
    return [
      createElement(Fragment, { key: "header" }, props.ListHeaderComponent as never),
      ...props.data.map((item) => createElement(Fragment, { key: props.keyExtractor(item) }, props.renderItem({ item }) as never)),
    ];
  },
}));

notifyManager.setScheduler((callback) => callback());

const TICKETS = 200;
const writer: MyWorkspaceInfo = { role_name: "Member", permissions: ["tickets:read", "tickets:write"] };
const statuses = [
  { id: "st-todo", name: "Todo", position: 0, kind: "backlog" },
  { id: "st-doing", name: "Doing", position: 1, kind: "progress" },
  { id: "st-shipped", name: "Shipped", position: 2, kind: "done" },
];
const pool = ["frontend", "backend", "docs", "infra", "design", "bug", "chore", "ops"];

let labelReads = 0;
let titles: string[] = [];

// A read of a ticket's labels is one step of a scan over the board, so the counter sizes the work a frame costs.
const seed = () =>
  Array.from({ length: TICKETS }, (_, i) => {
    const ticket = {
      id: `t-${i}`,
      project_id: "p-1",
      type_id: "",
      title: titles[i] ?? `Ticket ${i}`,
      status: statuses[i % 3]!.id,
      position: i,
      number: i,
      developer: "",
      tester: "",
    };
    return Object.defineProperty(ticket, "labels", {
      enumerable: true,
      get: () => {
        labelReads++;
        return [pool[i % pool.length], pool[(i + 3) % pool.length]];
      },
    });
  });

const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1" }]);
  if (url.startsWith("/api/auth/me")) return Promise.resolve({ user: { id: "u1", login: "alice", name: "Alice" } });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve(writer);
  if (url.startsWith("/api/projects")) return Promise.resolve([{ id: "p-1", name: "Acme", prefix: "ACM" }]);
  if (url.startsWith("/api/statuses")) return Promise.resolve(statuses);
  if (url.startsWith("/api/tickets")) return Promise.resolve(seed());
  if (url.startsWith("/api/ticket-types")) return Promise.resolve([]);
  throw new Error(`unexpected GET ${url}`);
};

beforeEach(() => {
  titles = [];
  labelReads = 0;
  useBoardStore.setState({ selectedProjectId: null });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(mockGet);
  jest.mocked(api.post).mockReset().mockResolvedValue({});
});

test("one ticket.updated frame re-renders the changed card and scans the board's labels once, not once per card", async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  await render(
    <QueryClientProvider client={client}>
      <BoardScreen />
    </QueryClientProvider>,
  );
  await screen.findByText("Ticket 0");
  await act(async () => {});

  const cardRenders = jest.mocked(useTicketPersonLookup);
  const rendersBefore = cardRenders.mock.calls.length;
  labelReads = 0;
  titles[0] = "Ticket 0, renamed";
  // What the live dispatcher does for the frame: refetch the project's ticket list.
  await act(() => client.invalidateQueries({ queryKey: [getTicketsByProjectKey] }));
  await screen.findByText("Ticket 0, renamed");

  const commits = cardRenders.mock.calls.length - rendersBefore;
  expect(commits).toBeLessThanOrEqual(2);
  expect(labelReads).toBeLessThanOrEqual(4 * TICKETS);
});
