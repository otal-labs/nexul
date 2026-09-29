import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";
import type { ReactElement } from "react";

import { api } from "@/api/client";
import { InboxScreen } from "@/components/inbox/InboxScreen";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
  errorMessage: jest.fn(() => "Something went wrong"),
}));

const mockPush = jest.fn();
const mockSetOptions = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
  useNavigation: () => ({ setOptions: mockSetOptions }),
}));

const ticketNotification = {
  id: "n1",
  user_id: "u1",
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
  kind: "doc.updated",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: true,
  created_at: "2026-08-02T11:00:00Z",
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InboxScreen />
    </QueryClientProvider>,
  );
};

// The header's "Mark all read" button is handed to the navigator via setOptions, never rendered in this
// tree, so a test grabs the latest headerRight the screen registered and calls its onPress directly.
type HeaderButton = ReactElement<{ onPress: () => void }>;

const lastHeaderButton = (): HeaderButton | undefined => {
  const call = mockSetOptions.mock.calls.at(-1) as [{ headerRight?: () => HeaderButton | undefined }];
  return call[0].headerRight?.();
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.post).mockReset();
  mockPush.mockReset();
  mockSetOptions.mockReset();
});

describe("InboxScreen", () => {
  test("renders notifications newest first, tap marks read and opens the subject", async () => {
    jest.mocked(api.get).mockResolvedValue([ticketNotification, docNotification]);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();

    const titles = await screen.findAllByText(/Write migrations|Spec/);
    expect(titles.map((node) => node.props.children)).toEqual(["Write migrations", "Spec"]);
    expect(screen.getByLabelText("Unread")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: /Write migrations/ }));

    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
    expect(mockPush).toHaveBeenCalledWith("/board/ticket/t-1", { withAnchor: true });
  });

  test("tapping an already-read row opens its subject without marking read again", async () => {
    jest.mocked(api.get).mockResolvedValue([ticketNotification, docNotification]);
    await renderScreen();
    await screen.findByText("Spec");

    await userEvent.setup().press(screen.getByRole("button", { name: /Spec/ }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-1", { withAnchor: true });
  });

  test("shows the empty state when there are no notifications", async () => {
    jest.mocked(api.get).mockResolvedValue([]);
    await renderScreen();

    expect(await screen.findByText("No notifications yet.")).toBeTruthy();
  });

  test("mark all read posts the read-all endpoint", async () => {
    jest.mocked(api.get).mockResolvedValue([ticketNotification]);
    jest.mocked(api.post).mockResolvedValue(undefined);
    await renderScreen();
    await screen.findByText("Write migrations");

    const headerButton = lastHeaderButton();
    headerButton?.props.onPress();

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all"));
  });
});
