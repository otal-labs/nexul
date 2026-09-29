import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { StackScreen } from "@/components/deploys/StackScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

jest.mock("expo-router", () => ({
  Stack: { Screen: () => null },
  useRouter: () => ({ push: jest.fn() }),
  useLocalSearchParams: () => ({ id: "s-1" }),
}));

const stack = { id: "s-1", project_id: "p-1", name: "api", machine: "hetzner-1", strategy: "run", managed: true };

const renderWithLatest = (status: string) => {
  jest.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/stacks/s-1") return Promise.resolve(stack);
    if (url === "/api/stacks/s-1/services") return Promise.resolve([]);
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

describe("StackScreen", () => {
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
