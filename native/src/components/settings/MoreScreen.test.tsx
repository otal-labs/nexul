import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderRouter, screen, waitFor } from "expo-router/testing-library";

import { api } from "@/api/client";
import { MoreScreen } from "@/components/settings/MoreScreen";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderMore = () =>
  renderRouter(
    {
      "(tabs)/more/index": () => (
        <QueryClientProvider client={client}>
          <MoreScreen />
        </QueryClientProvider>
      ),
    },
    { initialUrl: "/more" },
  );

const serve = (runners: unknown[], me: object) =>
  jest.mocked(api.get).mockImplementation(async (path: string) => {
    if (path === "/api/runners") return runners;
    if (path === "/api/auth/me") return { user: { id: "u-1" }, ...me };
    return [];
  });

describe("MoreScreen", () => {
  beforeEach(() => client.clear());

  test.each([
    { who: "a Client who sees no runner and can't add a machine", runners: [], me: { instance_permissions: [] }, row: false },
    { who: "someone who sees a runner", runners: [{ id: "r-1" }], me: { instance_permissions: [] }, row: true },
    { who: "someone who can add machines", runners: [], me: { instance_permissions: ["machines:create"] }, row: true },
  ])("Runners row for $who: $row", async ({ runners, me, row }) => {
    serve(runners, me);
    await renderMore();

    expect(await screen.findByText("Docs")).toBeTruthy();
    await waitFor(() => expect(client.getQueryState(["getRunners"])?.status).toBe("success"));
    await waitFor(() => expect(client.getQueryState(["getMe"])?.status).toBe("success"));
    await waitFor(() => expect(screen.queryByText("Runners") !== null).toBe(row));
  });
});
