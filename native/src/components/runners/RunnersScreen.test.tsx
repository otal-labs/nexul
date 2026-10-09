import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { RunnersScreen } from "@/components/runners/RunnersScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const runners = [
  { id: "r-1", name: "build-box", connected: true, last_seen: "2026-09-29T10:00:00Z", version: "v0.2.0", machine: "hetzner-1", running_job: null },
  { id: "r-2", name: "spare-box", connected: false, last_seen: "2026-09-20T10:00:00Z", version: "v0.1.9", running_job: null },
];

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RunnersScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => jest.mocked(api.get).mockReset());

describe("RunnersScreen", () => {
  test("renders each runner with its name, machine, and version", async () => {
    jest.mocked(api.get).mockResolvedValue(runners);
    await renderScreen();

    expect(await screen.findByText("build-box")).toBeTruthy();
    expect(screen.getByText("hetzner-1")).toBeTruthy();
    expect(screen.getByText("v0.2.0")).toBeTruthy();
    expect(screen.getByText("spare-box")).toBeTruthy();
    expect(screen.getByText("v0.1.9")).toBeTruthy();
  });

  test("shows the empty state when there are no runners", async () => {
    jest.mocked(api.get).mockResolvedValue([]);
    await renderScreen();

    expect(await screen.findByText("No runners yet")).toBeTruthy();
  });
});
