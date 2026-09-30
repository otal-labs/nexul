import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { StackScreen } from "@/components/deploys/StackScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

jest.mock("lucide-react-native", () => ({ ChevronRight: () => null }));

const mockPush = jest.fn();
const mockAccess: { current: boolean } = { current: true };
jest.mock("@/hooks/WorkspaceHooks", () => ({
  ...jest.requireActual<object>("@/hooks/WorkspaceHooks"),
  useAreaAccess: () => () => mockAccess.current,
}));

jest.mock("expo-router", () => ({
  Stack: { Screen: () => null },
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ id: "s-1" }),
}));

const stack = { id: "s-1", project_id: "p-1", name: "api", machine: "hetzner-1", strategy: "run", managed: true };

const web = { id: "c-1", stack_id: "s-1", name: "web", declared: { image: "nginx" }, status: "running" };

const renderWithLatest = (status: string, services: unknown[] = []) => {
  jest.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/stacks/s-1") return Promise.resolve(stack);
    if (url === "/api/stacks/s-1/services") return Promise.resolve(services);
    return Promise.resolve([
      { id: "d-1", stack_id: "s-1", image: "api:1", status: "healthy", created_at: "2026-09-28T10:00:00Z" },
      { id: "d-2", stack_id: "s-1", image: "api:2", status, created_at: "2026-09-29T10:00:00Z" },
    ]);
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <StackScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockPush.mockReset();
  mockAccess.current = true;
});

describe("StackScreen", () => {
  test("a service row opens that service's logs for a viewer who can read them", async () => {
    await renderWithLatest("healthy", [web]);

    await userEvent.press(await screen.findByRole("button", { name: "Logs for web" }));

    expect(mockPush).toHaveBeenCalledWith({ pathname: "/deploys/stack/[id]/logs/[service]", params: { id: "s-1", service: "web" } });
  });

  test("a service row is plain, with no logs affordance, without the permission", async () => {
    mockAccess.current = false;
    await renderWithLatest("healthy", [web]);

    expect(await screen.findByText("web")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Logs for web" })).toBeNull();
  });

  test.each(["pending", "running"])("Redeploy is held back while the latest deploy is %s", async (status) => {
    await renderWithLatest(status);

    const button = await screen.findByRole("button", { name: "Deploy in progress" });
    expect(button).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Redeploy" })).toBeNull();
  });

  test("Redeploy is offered once the latest deploy has finished", async () => {
    await renderWithLatest("failed");

    expect(await screen.findByRole("button", { name: "Redeploy" })).toBeEnabled();
  });
});
