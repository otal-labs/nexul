import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { BoardScreen } from "@/components/board/BoardScreen";
import type { Project } from "@/models/Project";
import type { MyWorkspaceInfo } from "@/models/Workspace";
import { useBoardStore } from "@/stores/boardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), patch: jest.fn() } }));

jest.mock("expo-router", () => ({ useRouter: () => ({ push: jest.fn() }) }));

jest.mock("expo-sqlite/kv-store", () => ({
  __esModule: true,
  default: { getItemSync: () => null, setItemSync: jest.fn(), removeItemSync: jest.fn() },
}));

const me = { user: { id: "u1", login: "onik97", name: "Onik" } };
const project = { id: "p-1", name: "Nexul", prefix: "NEX" };
const other = { id: "p-2", name: "Website", prefix: "WEB" };
let projects: Project[] = [project];
let myRole: MyWorkspaceInfo = { role_name: "Member", permissions: ["tickets:read"] };
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
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve(myRole);
  if (url.startsWith("/api/projects")) return Promise.resolve(projects);
  if (url.startsWith("/api/statuses")) return Promise.resolve(statuses);
  if (url.startsWith("/api/tickets")) return Promise.resolve(tickets);
  if (url.startsWith("/api/ticket-types")) return Promise.resolve([]);
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <BoardScreen />
    </QueryClientProvider>,
  );
  return client;
};

beforeEach(() => {
  projects = [project];
  myRole = { role_name: "Member", permissions: ["tickets:read"] };
  useBoardStore.setState({ selectedProjectId: null });
  jest.mocked(api.get).mockReset().mockImplementation(mockGet);
});


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

  test("a card's Move to action changes its column, the same move a drop makes", async () => {
    jest.mocked(api.patch).mockResolvedValue({});
    await renderScreen();
    const card = await screen.findByRole("button", { name: "NEX-12 Fix login" });

    expect(card.props.accessibilityActions.map((a: { label: string }) => a.label)).toEqual(["Move to Doing", "Move to Shipped"]);
    await act(() => fireEvent(card, "accessibilityAction", { nativeEvent: { actionName: "st-shipped" } }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "st-shipped" });
  });

  test("Only mine hides tickets where the viewer holds neither role", async () => {
    await renderScreen();
    await screen.findByText("Fix login");
    expect(screen.getByText("Add board")).toBeTruthy();
    expect(screen.getByText("Ship it")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("switch", { name: "Only my tickets" }));

    expect(screen.getByText("Fix login")).toBeTruthy();
    expect(screen.getByText("Add board")).toBeTruthy();
    expect(screen.queryByText("Ship it")).toBeNull();
  });

  test("a picked project taken away while open turns revoked instead of showing another project's board", async () => {
    projects = [project, other];
    useBoardStore.setState({ selectedProjectId: "p-1" });
    const client = await renderScreen();
    await screen.findByText("Fix login");
    expect(screen.getByText("Nexul")).toBeTruthy();

    projects = [other];
    await act(() => client.invalidateQueries());

    expect(await screen.findByText("You no longer have access to this project")).toBeTruthy();
    expect(screen.getByText("Board")).toBeTruthy();
    expect(screen.queryByText("Website")).toBeNull();
    expect(screen.queryByText("Fix login")).toBeNull();
  });

  test("a remembered pick the viewer never saw here falls back to the first project, not the revoked state", async () => {
    useBoardStore.setState({ selectedProjectId: "p-gone" });
    await renderScreen();

    expect(await screen.findByText("Fix login")).toBeTruthy();
    expect(screen.getByText("Nexul")).toBeTruthy();
    expect(screen.queryByText("You no longer have access to this project")).toBeNull();
  });

  test.each([
    { picked: "p-1", shows: "Fix login" },
    { picked: "p-2", shows: "This page doesn't exist" },
  ])("a Restricted member's board on $picked follows that project's own access", async ({ picked, shows }) => {
    projects = [project, other];
    myRole = {
      role_name: "Member",
      permissions: [],
      restricted: true,
      projects: [
        { project_id: "p-1", actions: ["tickets:read"] },
        { project_id: "p-2", actions: ["docs:read"] },
      ],
    };
    useBoardStore.setState({ selectedProjectId: picked });
    await renderScreen();

    expect(await screen.findByText(shows)).toBeTruthy();
  });
});
