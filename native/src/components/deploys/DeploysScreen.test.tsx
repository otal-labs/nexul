import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { DeploysScreen } from "@/components/deploys/DeploysScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

const stacks = [
  { id: "s-1", project_id: "p-1", name: "api", machine: "hetzner-1", strategy: "run", managed: true },
  { id: "s-2", project_id: "p-1", name: "worker", machine: "hetzner-2", strategy: "run", managed: true },
];

const deploysByStack: Record<string, unknown[]> = {
  "s-1": [
    { id: "d-1", stack_id: "s-1", image: "api:1", status: "healthy", created_at: "2026-09-28T10:00:00Z" },
    { id: "d-2", stack_id: "s-1", image: "api:2", status: "failed", created_at: "2026-09-29T10:00:00Z" },
  ],
  "s-2": [],
};

// Stacks answer only when scoped to the selected workspace, so the list never shows another workspace's.
const mockGet = (url: string) => {
  if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
  if (url === "/api/stacks?workspace_id=ws-1") return Promise.resolve(stacks);
  const match = /\/api\/stacks\/(.+)\/deploys/.exec(url);
  if (match?.[1]) return Promise.resolve(deploysByStack[match[1]] ?? []);
  throw new Error(`unexpected GET ${url}`);
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DeploysScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(mockGet);
  mockPush.mockReset();
});

describe("DeploysScreen", () => {
  test("renders each stack with its name, target, and last-deploy time", async () => {
    await renderScreen();

    expect(await screen.findByText("api")).toBeTruthy();
    expect(screen.getByText("hetzner-1")).toBeTruthy();
    expect(screen.getByText("worker")).toBeTruthy();
    expect(screen.getByText("hetzner-2")).toBeTruthy();
    // s-1's latest deploy is d-2 (newer created_at), so its image never appears in this list row.
    expect(screen.queryByText("api:1")).toBeNull();
  });

  test("a stack with no deploys yet shows the no-deploys meta instead of a time", async () => {
    await renderScreen();

    await screen.findByText("worker");
    expect(screen.getByText("No deploys")).toBeTruthy();
  });

  test("shows the empty state when there are no stacks", async () => {
    jest.mocked(api.get).mockImplementation((url: string) => (url.startsWith("/api/stacks?") ? Promise.resolve([]) : mockGet(url)));
    await renderScreen();

    expect(await screen.findByText("No stacks yet.")).toBeTruthy();
  });
});
