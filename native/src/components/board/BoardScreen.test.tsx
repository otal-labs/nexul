import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { BoardScreen } from "@/components/board/BoardScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

jest.mock("expo-router", () => ({ useRouter: () => ({ push: jest.fn() }) }));

// lucide-react-native ships ESM-only under the "react-native" package.json field, which this project's
// jest transform (`\.[jt]sx?$`) doesn't cover; a hand-written fake sidesteps that instead of rendering icons.
jest.mock("lucide-react-native", () => ({ ChevronDown: () => null }));

jest.mock("expo-sqlite/kv-store", () => ({
  __esModule: true,
  default: { getItemSync: () => null, setItemSync: jest.fn(), removeItemSync: jest.fn() },
}));

const me = { user: { id: "u1", login: "onik97", name: "Onik" } };
const project = { id: "p-1", name: "Nexul", prefix: "NEX" };
// Deliberately not alphabetical, so a test that passed by accidentally sorting statuses by name would fail here.
const statuses = [
  { id: "st-todo", name: "Todo", position: 0, kind: "backlog" },
  { id: "st-doing", name: "Doing", position: 0, kind: "progress" },
  { id: "st-shipped", name: "Shipped", position: 0, kind: "done" },
];
const tickets = [
  { id: "t-1", project_id: "p-1", type_id: "", title: "Fix login", status: "st-todo", position: 0, number: 12, developer: "onik97", tester: "", labels: null },
  { id: "t-2", project_id: "p-1", type_id: "", title: "Add board", status: "st-doing", position: 0, number: 13, developer: "someone-else", tester: "onik97", labels: null },
  { id: "t-3", project_id: "p-1", type_id: "", title: "Ship it", status: "st-shipped", position: 0, number: 14, developer: "", tester: "", labels: null },
];

const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1" }]);
  if (url.startsWith("/api/auth/me")) return Promise.resolve(me);
  if (url.startsWith("/api/projects")) return Promise.resolve([project]);
  if (url.startsWith("/api/statuses")) return Promise.resolve(statuses);
  if (url.startsWith("/api/tickets")) return Promise.resolve(tickets);
  if (url.startsWith("/api/ticket-types")) return Promise.resolve([]);
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <BoardScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => jest.mocked(api.get).mockReset().mockImplementation(mockGet));


beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("BoardScreen", () => {
  test("sections render in the project's status order", async () => {
    await renderScreen();
    await screen.findByText("Fix login");

    const tree = JSON.stringify(screen.toJSON());
    const todo = tree.indexOf("Todo");
    const doing = tree.indexOf("Doing");
    const shipped = tree.indexOf("Shipped");
    expect(todo).toBeGreaterThan(-1);
    expect(todo).toBeLessThan(doing);
    expect(doing).toBeLessThan(shipped);
  });

  test("Mine hides tickets where the viewer holds neither role", async () => {
    await renderScreen();
    await screen.findByText("Fix login");
    expect(screen.getByText("Add board")).toBeTruthy();
    expect(screen.getByText("Ship it")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: "Mine" }));

    expect(screen.getByText("Fix login")).toBeTruthy();
    expect(screen.getByText("Add board")).toBeTruthy();
    expect(screen.queryByText("Ship it")).toBeNull();
  });
});
