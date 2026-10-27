import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { SearchScreen } from "@/components/search/SearchScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
}));

const hit = (type: string, id: string, title: string, extra: Record<string, unknown> = {}) => ({ type, id, title, in_body: false, updated_at: "", ...extra });

const mockSearch = (answer: () => Promise<unknown>) => {
  jest.mocked(api.get).mockImplementation((path: string) => {
    if (path === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
    if (path.startsWith("/api/search?")) return answer();
    return Promise.resolve([]);
  });
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SearchScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  mockPush.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("SearchScreen", () => {
  test("a failed search shows the error", async () => {
    mockSearch(() => Promise.reject(new Error("Search is down")));
    await renderScreen();
    await userEvent.setup().type(screen.getByLabelText("Search this workspace"), "checkout");
    expect(await screen.findByText(/Search is down/)).toBeTruthy();
  });

  test("a query nothing matches says so", async () => {
    mockSearch(() => Promise.resolve({ hits: [hit("memory", "m-1", "Checkout runbook")], more: [] }));
    await renderScreen();
    await userEvent.setup().type(screen.getByLabelText("Search this workspace"), "checkout");
    expect(await screen.findByText("No results")).toBeTruthy();
    expect(screen.queryByText("Checkout runbook")).toBeNull();
  });

  test("groups the hits the phone can open, keeps text-only hits apart, and opens a pull request's ticket", async () => {
    mockSearch(() =>
      Promise.resolve({
        hits: [
          hit("ticket", "t-1", "Checkout flow", { hint: "BE-4" }),
          hit("ticket", "t-2", "Payments", { hint: "BE-5", in_body: true }),
          hit("pull_request", "acme/app#9", "Speed up checkout", { hint: "app#9", ticket_id: "t-1" }),
          hit("channel", "c-1", "checkout"),
          hit("automation", "a-1", "Checkout alerts"),
        ],
        more: ["ticket"],
      }),
    );
    await renderScreen();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText("Search this workspace"), "checkout");

    expect(await screen.findByText("Checkout flow")).toBeTruthy();
    expect(screen.getByText("In text")).toBeTruthy();
    expect(screen.getByText("#checkout")).toBeTruthy();
    expect(screen.queryByText("Checkout alerts")).toBeNull();
    expect(screen.getByText("More matches than shown. Add a word to narrow.")).toBeTruthy();
    expect(jest.mocked(api.get)).toHaveBeenCalledWith("/api/search?limit=5&workspace_id=ws-1&q=checkout");

    await user.press(screen.getByRole("button", { name: /Speed up checkout/ }));
    expect(mockPush).toHaveBeenCalledWith("/board/ticket/t-1", { withAnchor: true });
  });
});
