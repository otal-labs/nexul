import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { DocsListScreen } from "@/components/docs/DocsListScreen";
import { useDocsProjectStore } from "@/stores/docsProjectStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
}));

const workspace = { id: "ws-1", name: "Acme" };
const projectA = { id: "proj-a", name: "Alpha", prefix: "AL" };
const projectB = { id: "proj-b", name: "Beta", prefix: "BE" };

const docA = {
  id: "doc-a",
  project_id: "proj-a",
  title: "Alpha spec",
  version: 1,
  archived: false,
  can_open: true,
  updated_at: new Date().toISOString(),
};

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DocsListScreen />
    </QueryClientProvider>,
  );
};

const mockGet = (docsByProject: Record<string, unknown>, projects = [projectA, projectB]) => {
  jest.mocked(api.get).mockImplementation((path: string) => {
    if (path === "/api/workspaces") return Promise.resolve([workspace]);
    if (path === `/api/projects?workspace_id=${workspace.id}`) return Promise.resolve(projects);
    const match = /project_id=([^&]+)/.exec(path);
    return Promise.resolve(match ? docsByProject[match[1] ?? ""] ?? [] : []);
  });
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  mockPush.mockReset();
  useDocsProjectStore.setState({ selectedProjectId: null });
});


beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("DocsListScreen", () => {
  test("lists the first project's docs and opens a doc on tap", async () => {
    mockGet({ [projectA.id]: [docA] });
    await renderScreen();

    expect(await screen.findByText("Alpha")).toBeTruthy();
    expect(await screen.findByText("Alpha spec")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: /Alpha spec/ }));
    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-a");
  });

  test("switching the selected project loads that project's docs", async () => {
    const docB = { ...docA, id: "doc-b", project_id: "proj-b", title: "Beta spec" };
    mockGet({ [projectA.id]: [docA], [projectB.id]: [docB] });
    await renderScreen();
    await screen.findByText("Alpha spec");

    await act(async () => useDocsProjectStore.getState().setSelectedProjectId(projectB.id));

    expect(await screen.findByText("Beta spec")).toBeTruthy();
    expect(screen.queryByText("Alpha spec")).toBeNull();
  });

  test("a project picked in another workspace gives way to this workspace's first project", async () => {
    const docOther = { ...docA, id: "doc-o", project_id: "proj-other", title: "Other workspace spec" };
    mockGet({ [projectA.id]: [docA], "proj-other": [docOther] });
    useDocsProjectStore.setState({ selectedProjectId: "proj-other" });
    await renderScreen();

    expect(await screen.findByText("Alpha spec")).toBeTruthy();
    expect(screen.queryByText("Other workspace spec")).toBeNull();
  });

  test("shows the empty state when the project has no docs", async () => {
    mockGet({ [projectA.id]: [] });
    await renderScreen();

    expect(await screen.findByText("No docs yet")).toBeTruthy();
  });
});
