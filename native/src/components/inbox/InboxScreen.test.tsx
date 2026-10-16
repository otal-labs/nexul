import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import { api } from "@/api/client";
import { InboxScreen } from "@/components/inbox/InboxScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
}));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
}));

const ticketNotification = {
  id: "n1",
  user_id: "u1",
  workspace_id: "ws-1",
  kind: "ticket.assigned",
  subject_type: "ticket",
  subject_id: "t-1",
  subject_title: "Write migrations",
  read: false,
  created_at: "2026-08-12T12:00:00Z",
};

const docNotification = {
  id: "n2",
  user_id: "u1",
  workspace_id: "ws-1",
  kind: "doc.updated",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: true,
  created_at: "2026-08-02T11:00:00Z",
};

// Serves the selected workspace's inbox; any other path is a request the screen should not make.
const mockInbox = (notifications: unknown[], permissions: string[] = ["tickets:read"]) =>
  jest.mocked(api.get).mockImplementation((path: string) => {
    if (path === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
    if (path === "/api/workspaces/ws-1/me") return Promise.resolve({ role_name: "Member", permissions });
    if (path === "/api/notifications?workspace_id=ws-1") return Promise.resolve(notifications);
    return Promise.reject(new Error(`unexpected GET ${path}`));
  });

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InboxScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.post).mockReset();
  mockPush.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("InboxScreen", () => {
  test("renders notifications newest first, tap marks read and opens the subject", async () => {
    mockInbox([ticketNotification, docNotification]);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();

    const titles = await screen.findAllByText(/Write migrations|Spec/);
    expect(titles.map((node) => node.props.children)).toEqual(["Write migrations", "Spec"]);
    expect(screen.getByLabelText("Unread")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: /Write migrations/ }));

    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
    expect(mockPush).toHaveBeenCalledWith("/board/ticket/t-1", { withAnchor: true });
  });

  test.each([
    ["an unread row", /Write migrations/, "Mark read", "markRead", "/api/notifications/n1/read"],
    ["a read row", /Spec/, "Mark unread", "markUnread", "/api/notifications/n2/unread"],
  ])("%s offers one flip, the swipe's alternative, and makes it without opening the row", async (_name, title, label, actionName, path) => {
    mockInbox([ticketNotification, docNotification]);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();
    const row = await screen.findByRole("button", { name: title });

    expect(row.props.accessibilityActions).toEqual([{ name: actionName, label }]);
    await act(() => fireEvent(row, "accessibilityAction", { nativeEvent: { actionName } }));

    expect(api.post).toHaveBeenCalledWith(path);
    expect(mockPush).not.toHaveBeenCalled();
  });

  test("tapping an already-read row opens its subject without marking read again", async () => {
    mockInbox([ticketNotification, docNotification]);
    await renderScreen();
    await screen.findByText("Spec");

    await userEvent.setup().press(screen.getByRole("button", { name: /Spec/ }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-1", { withAnchor: true });
  });

  test.each(["doc.questions_asked", "doc.questions_answered"])("tapping a %s notice opens the doc on its questions", async (kind) => {
    mockInbox([{ ...docNotification, kind }]);
    await renderScreen();
    await screen.findByText("Spec");

    await userEvent.setup().press(screen.getByRole("button", { name: /Spec/ }));

    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-1?view=questions", { withAnchor: true });
  });

  test("tapping a crashed-service notice opens the stack's screen", async () => {
    mockInbox([{ ...docNotification, kind: "service.container_stopped", subject_type: "stack", subject_id: "stack-1", subject_title: "api crashed on web: exit 1" }]);
    await renderScreen();
    await screen.findByText("api crashed on web: exit 1");

    await userEvent.setup().press(screen.getByRole("button", { name: /api crashed/ }));

    expect(screen.getByText("Service crashed")).toBeTruthy();
    expect(mockPush).toHaveBeenCalledWith("/deploys/stack/stack-1", { withAnchor: true });
  });

  test("tapping an error-rate notice opens the stack's screen", async () => {
    mockInbox([{ ...docNotification, kind: "stack.error_rate_crossed", subject_type: "stack", subject_id: "stack-1", subject_title: "5xx at 12% on web over the last 5 minutes, above 5%" }]);
    await renderScreen();
    await screen.findByText(/5xx at 12% on web/);

    await userEvent.setup().press(screen.getByRole("button", { name: /5xx at 12%/ }));

    expect(screen.getByText("5xx over the limit")).toBeTruthy();
    expect(mockPush).toHaveBeenCalledWith("/deploys/stack/stack-1", { withAnchor: true });
  });

  test("tapping a ticket notification marks it read but stays in the inbox without tickets:read", async () => {
    mockInbox([ticketNotification], []);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();
    await screen.findByText("Write migrations");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));

    await userEvent.setup().press(screen.getByRole("button", { name: /Write migrations/ }));

    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
    expect(mockPush).not.toHaveBeenCalled();
  });

  test("shows the empty state when there are no notifications", async () => {
    mockInbox([]);
    await renderScreen();

    expect(await screen.findByText("Nothing needs you")).toBeTruthy();
  });

  test("mark all read marks only the selected workspace read", async () => {
    mockInbox([ticketNotification]);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();
    await screen.findByText("Write migrations");

    await userEvent.setup().press(screen.getByRole("button", { name: "Mark all read" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all?workspace_id=ws-1"));
  });
});
