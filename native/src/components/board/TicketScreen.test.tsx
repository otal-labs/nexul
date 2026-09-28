import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { TicketScreen } from "@/components/board/TicketScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), patch: jest.fn(), post: jest.fn() } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ id: "t-1" }),
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
  developer: "onik97",
  tester: "",
  labels: null,
};

const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1" }]);
  if (url.startsWith("/api/auth/me")) return Promise.resolve(me);
  if (url === "/api/tickets/t-1") return Promise.resolve(ticket);
  if (url.startsWith("/api/projects/")) return Promise.resolve(project);
  if (url.startsWith("/api/statuses")) return Promise.resolve(statuses);
  if (url.startsWith("/api/ticket-types")) return Promise.resolve(ticketTypes);
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TicketScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
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
    expect(screen.getByText("onik97")).toBeTruthy();
    expect(screen.getByText("No one")).toBeTruthy();
  });

  test("Assign to me sets the viewer as developer", async () => {
    jest.mocked(api.patch).mockResolvedValue({ ...ticket, developer: "onik97" });
    await renderScreen();
    await screen.findByText("Fix login");

    await userEvent.setup().press(screen.getByRole("button", { name: "Assign to me" }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/developer", { login: "onik97" });
  });
});
