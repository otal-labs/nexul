import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { TicketScreen } from "@/components/board/TicketScreen";
import { getProjectsKey } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The shared renderer pulls in the app query client and native markdown; the ticket screen only needs the body text.
jest.mock("@/components/chat/MessageBody", () => {
  const { Text } = jest.requireActual("react-native");
  return { MessageBody: ({ body }: { body: string }) => <Text>{body}</Text> };
});
jest.mock("@/api/client", () => ({ api: { get: jest.fn(), patch: jest.fn(), post: jest.fn() } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ id: "t-1" }),
  Stack: { Screen: () => null },
}));

const me = { user: { id: "u1", login: "onik97", name: "Onik" } };
const project = { id: "p-1", name: "Nexul", prefix: "NEX" };
const statuses = [{ id: "st-open", name: "Open", position: 0, kind: "backlog" }];
const ticketTypes = [{ id: "ty-1", name: "bug", color: "" }];
const ticket = {
  id: "t-1",
  project_id: "p-1",
  type_id: "ty-1",
  title: "Fix login",
  body: "Steps to repro the bug.",
  status: "st-open",
  position: 0,
  number: 12,
  developer: "",
  tester: "",
  labels: null,
};
let currentTicket: typeof ticket = ticket;
let myRole: MyWorkspaceInfo;
notifyManager.setScheduler((callback) => callback());

const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1" }]);
  if (url.startsWith("/api/auth/me")) return Promise.resolve(me);
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve(myRole);
  if (url === "/api/tickets/t-1") return Promise.resolve(currentTicket);
  if (url.startsWith("/api/projects/")) return Promise.resolve(project);
  if (url.startsWith("/api/statuses")) return Promise.resolve(statuses);
  if (url.startsWith("/api/ticket-types")) return Promise.resolve(ticketTypes);
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <TicketScreen />
    </QueryClientProvider>,
  );
  return client;
};

beforeEach(() => {
  currentTicket = ticket;
  myRole = { role_name: "Developer", permissions: ["tickets:write"], show_locked_actions: false };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(mockGet);
  jest.mocked(api.patch).mockReset();
  jest.mocked(api.post).mockReset();
  mockPush.mockReset();
});

describe("TicketScreen", () => {
  test("renders the ticket's id, title, status, type, and assignees", async () => {
    await renderScreen();

    await screen.findByText("Fix login");
    expect(screen.getByText("NEX-12")).toBeTruthy();
    expect(screen.getByText("Open")).toBeTruthy();
    expect(screen.getByText("bug")).toBeTruthy();
    expect(screen.getAllByText("No one")).toHaveLength(2);
  });

  test.each([
    ["hides both writes from a read-only role", { permissions: ["tickets:read"], show_locked_actions: false }, undefined, undefined],
    [
      "locks both writes for a read-only role that shows locked actions",
      { permissions: ["tickets:read"], show_locked_actions: true },
      "Your role can't change a ticket's status",
      "Your role can't assign tickets",
    ],
  ])("%s", async (_name, role, statusHint, assignHint) => {
    myRole = { role_name: "Client", ...role };
    await renderScreen();
    await screen.findByText("Fix login");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await act(async () => {});

    const status = screen.queryByRole("button", { name: /^Status, Open/ });
    const assign = screen.queryByRole("button", { name: "Assign to me" });
    expect(status === null).toBe(statusHint === undefined);
    expect(assign === null).toBe(assignHint === undefined);
    expect(status?.props.accessibilityHint).toBe(statusHint);
    expect(assign?.props.accessibilityHint).toBe(assignHint);
    expect(screen.getByText("Open")).toBeTruthy();
  });

  test("Assign to me sets the viewer as developer", async () => {
    jest.mocked(api.patch).mockResolvedValue({ ...ticket, developer: "onik97" });
    await renderScreen();
    await screen.findByText("Fix login");

    await userEvent.setup().press(screen.getByRole("button", { name: "Assign to me" }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/developer", { login: "onik97" });
  });

  test("Assign to me is hidden when the viewer is already the developer", async () => {
    currentTicket = { ...ticket, developer: "onik97" };
    await renderScreen();
    await screen.findByText("Fix login");
    await screen.findByText("onik97");

    await waitFor(() => expect(screen.queryByRole("button", { name: "Assign to me" })).toBeNull());
  });

  test("a ticket whose project is taken away while open turns revoked, not deleted", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    let revoked = false;
    jest.mocked(api.get).mockImplementation(async (url: string) => {
      if (url.startsWith("/api/projects?")) return revoked ? [] : [project];
      if (url === "/api/tickets/t-1" && revoked) throw new ApiError(404, { message: "not found" }, "GET failed: 404");
      return mockGet(url);
    });
    const client = await renderScreen();
    await screen.findByText("Fix login");
    await waitFor(() => expect(client.getQueryData([getProjectsKey, "ws-1"])).toEqual([project]));

    revoked = true;
    await act(() => client.invalidateQueries());

    expect(await screen.findByText("You no longer have access to this project")).toBeTruthy();
    expect(screen.queryByText("This ticket doesn't exist or was deleted.")).toBeNull();
    expect(screen.queryByText("Fix login")).toBeNull();
  });
});
