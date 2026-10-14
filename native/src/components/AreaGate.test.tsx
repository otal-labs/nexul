import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Slot, Stack } from "expo-router";
import { renderRouter, screen } from "expo-router/testing-library";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import BoardLayout from "@/app/(tabs)/board/_layout";
import TabsLayout from "@/app/(tabs)/_layout";
import { MoreScreen } from "@/components/settings/MoreScreen";
import { Text } from "@/components/ui/text";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

let permissions: string[] = [];
let projects: MyWorkspaceInfo["projects"];
let runners: { id: string }[] = [];

const renderApp = (initialUrl: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderRouter(
    {
      _layout: () => (
        <QueryClientProvider client={client}>
          <Slot />
        </QueryClientProvider>
      ),
      "(tabs)/_layout": TabsLayout,
      "(tabs)/inbox/_layout": () => <Stack />,
      "(tabs)/inbox/index": () => <Text>Inbox list</Text>,
      "(tabs)/chat/_layout": () => <Stack />,
      "(tabs)/chat/index": () => <Text>Chat list</Text>,
      "(tabs)/board/_layout": BoardLayout,
      "(tabs)/board/index": () => <Text>Board list</Text>,
      "(tabs)/deploys/_layout": () => <Stack />,
      "(tabs)/deploys/index": () => <Text>Deploys list</Text>,
      "(tabs)/more/_layout": () => <Stack />,
      "(tabs)/more/index": MoreScreen,
    },
    { initialUrl },
  );
};

beforeEach(() => {
  projects = undefined;
  runners = [];
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockImplementation(async (path: string) => {
    if (path === "/api/workspaces") return [{ id: "ws-1", name: "Acme" }];
    if (path === "/api/workspaces/ws-1/me") return { role_name: "Member", permissions, ...(projects && { projects }) };
    if (path === "/api/runners") return runners;
    if (path.startsWith("/api/notifications/unread-count")) return { count: 0 };
    return [];
  });
});

// A tab's accessible name reads "Board, tab, 3 of 5"; a tab dropped with href null is not rendered visible.
const tab = (name: string) => new RegExp(`^${name}, tab`);
const tabs = () => ["Inbox", "Chat", "Board", "Deploys", "More"].filter((name) => screen.queryByRole("button", { name: tab(name) }));

describe("tabs by permission", () => {
  test.each([
    { name: "the Owner's whole grid shows every tab", granted: ["tickets:read", "stacks:read"], shown: ["Inbox", "Chat", "Board", "Deploys", "More"] },
    { name: "tickets alone adds Board", granted: ["tickets:read"], shown: ["Inbox", "Chat", "Board", "More"] },
    { name: "stacks alone adds Deploys", granted: ["stacks:read"], shown: ["Inbox", "Chat", "Deploys", "More"] },
  ])("$name", async ({ granted, shown }) => {
    permissions = granted;
    await renderApp("/inbox");

    expect(await screen.findByRole("button", { name: tab(shown[2] ?? "") })).toBeTruthy();
    expect(tabs()).toEqual(shown);
  });

  test("a member who reads neither keeps only Inbox, Chat and More", async () => {
    permissions = [];
    await renderApp("/more");

    expect(await screen.findByText("Docs")).toBeTruthy();
    expect(tabs()).toEqual(["Inbox", "Chat", "More"]);
  });

  test("More lists Runners for a member who sees one, the server deciding which machines they see", async () => {
    permissions = ["tickets:read"];
    runners = [{ id: "r-1" }];
    await renderApp("/more");

    expect(await screen.findByRole("button", { name: tab("Board") })).toBeTruthy();
    expect(screen.getByText("Docs")).toBeTruthy();
    expect(screen.getByText("Runners")).toBeTruthy();
  });

  test("what a member holds in a project they are in opens its tab", async () => {
    permissions = ["chat:write"];
    projects = [{ project_id: "p-1", actions: ["projects:read", "tickets:read"] }];
    await renderApp("/board");

    expect(await screen.findByText("Board list")).toBeTruthy();
    expect(tabs()).toEqual(["Inbox", "Chat", "Board", "More"]);
  });

  test("a member no project lists keeps Inbox, Chat and More", async () => {
    permissions = ["chat:write"];
    projects = [];
    await renderApp("/chat");

    expect(await screen.findByText("Chat list")).toBeTruthy();
    expect(tabs()).toEqual(["Inbox", "Chat", "More"]);
  });

  test("a deep link into a tab the viewer can't read is not found", async () => {
    permissions = ["stacks:read"];
    await renderApp("/board");

    expect(await screen.findByText("This page doesn't exist")).toBeTruthy();
    expect(screen.queryByText("Board list")).toBeNull();
  });
});
