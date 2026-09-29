import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { router, Stack } from "expo-router";
import { Tabs } from "expo-router/js-tabs";
import { act, fireEvent, renderRouter, screen } from "expo-router/testing-library";

import { api } from "@/api/client";
import { WorkspaceScreen } from "@/components/settings/WorkspaceScreen";
import { Text } from "@/components/ui/text";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

// A real router with the app's tab and stack shape, so the test sees what a switch leaves on each tab.
const renderTabs = () =>
  renderRouter(
    {
      "(tabs)/_layout": () => <Tabs />,
      "(tabs)/board/_layout": () => <Stack />,
      "(tabs)/board/index": () => <Text>Board list</Text>,
      "(tabs)/board/ticket": () => <Text>Old workspace ticket</Text>,
      "(tabs)/more/_layout": () => <Stack />,
      "(tabs)/more/index": () => <Text>More list</Text>,
      "(tabs)/more/settings/_layout": () => <Stack />,
      "(tabs)/more/settings/index": () => <Text>Your settings</Text>,
      "(sheets)/more/settings/workspace": () => (
        <QueryClientProvider client={client}>
          <WorkspaceScreen />
        </QueryClientProvider>
      ),
    },
    { initialUrl: "/board/ticket" },
  );

// Navigation state lands on timers renderRouter fakes, so each step flushes them before the next.
const settle = async (step: () => void) => {
  act(step);
  await act(async () => jest.runAllTimers());
};

beforeEach(() => {
  client.clear();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockResolvedValue([
    { id: "ws-1", name: "Acme" },
    { id: "ws-2", name: "Beta" },
  ]);
});

describe("WorkspaceScreen", () => {
  test("switching workspace sends every tab back to its root", async () => {
    // renderRouter decorates the pending render, not what it resolves to, so the pathname reads from the promise.
    const view = renderTabs();
    await view;
    expect(screen.getByText("Old workspace ticket")).toBeTruthy();
    await settle(() => router.navigate("/more"));
    await settle(() => router.push("/more/settings"));
    await settle(() => router.push("/more/settings/workspace"));

    fireEvent.press(await screen.findByText("Beta"));
    await settle(() => undefined);

    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
    expect(view.getPathname()).toBe("/more");
    await settle(() => router.navigate("/board"));
    expect(screen.getByText("Board list")).toBeTruthy();
    expect(screen.queryByText("Old workspace ticket", { includeHiddenElements: true })).toBeNull();
  });
});
