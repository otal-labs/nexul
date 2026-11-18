import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ImportSection } from "@/components/import/ImportSection";
import { getImportsKey, importFollower } from "@/hooks/ImportHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Import } from "@/models/Import";
import { followFrame } from "@/test/followFrame";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const report = {
  tickets: 250,
  comments: 275,
  docs: 0,
  people: [
    { login: "alice", name: "alice", member: "alice" },
    { login: "zed", name: "zed", member: "" },
  ],
  statuses: [
    { name: "", kind: "backlog", column: "Backlog", new: false },
    { name: "", kind: "done", column: "Done", new: true },
  ],
  skipped: 0,
  files_copied: 0,
  files_linked: 0,
  unsupported: 0,
  relations_skipped: 0,
  failures: [],
};

const anImport = (over: Partial<Import> = {}): Import => ({
  id: "imp-1",
  workspace_id: "ws-1",
  project_id: "p-1",
  source: "github",
  from: "acme/app",
  state: "ready",
  created_by: "u-1",
  total: 250,
  done: 0,
  failed: 0,
  report,
  created_at: "2026-11-18T10:00:00Z",
  updated_at: "2026-11-18T10:00:00Z",
  ...over,
});

const renderSection = (
  imports: Import[],
  repos = [
    {
      owner: "acme",
      name: "app",
      full_name: "acme/app",
      connector_id: "github",
      role: "app",
    },
  ],
) => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/imports") return Promise.resolve({ data: imports });
    if (url === "/api/projects")
      return Promise.resolve({
        data: [{ id: "p-1", name: "Atlas", workspace_id: "ws-1" }],
      });
    if (url === "/api/projects/p-1/repos") return Promise.resolve({ data: repos });
    if (url.startsWith("/api/workspaces/ws-1/people")) return Promise.resolve({ data: [] });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 30_000 } },
  });
  render(
    <QueryClientProvider client={client}>
      <ImportSection />
    </QueryClientProvider>,
  );
  return client;
};

describe("ImportSection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("scans a Linear team with the API key it is given", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: anImport({ source: "linear", state: "scanning", total: 0 }) });
    renderSection([]);

    const scan = await screen.findByRole("button", { name: "Scan" });
    await pickOption(user, "Source", "Linear");
    await pickOption(user, "Project", "Atlas");
    expect(scan).toBeDisabled();
    await user.type(screen.getByLabelText("API key"), "lin_api_test");
    await user.type(screen.getByLabelText("Team"), "ENG");
    await user.click(scan);
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/imports", {
        source: "linear",
        project_id: "p-1",
        from: "ENG",
        credential: "lin_api_test",
      }),
    );
    expect(mocks.get).not.toHaveBeenCalledWith("/api/projects/p-1/repos");
  });

  it("scans the picked project's one repository", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({
      data: anImport({ state: "scanning", total: 0 }),
    });
    renderSection([]);

    const scan = await screen.findByRole("button", { name: "Scan" });
    expect(scan).toBeDisabled();
    await pickOption(user, "Project", "Atlas");
    expect(await screen.findByText("acme/app")).toBeInTheDocument();
    await user.click(scan);
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/imports", {
        source: "github",
        project_id: "p-1",
        from: "acme/app",
      }),
    );
    expect(await screen.findByText("Scanning")).toBeInTheDocument();
  });

  it("scans Notion with the integration secret and the root pages", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: anImport({ source: "notion", state: "scanning", total: 0 }) });
    renderSection([]);

    await pickOption(user, "Source", "Notion pages");
    await pickOption(user, "Project", "Atlas");
    const scan = screen.getByRole("button", { name: "Scan" });
    expect(scan).toBeDisabled();
    await user.type(screen.getByLabelText("Integration secret"), "ntn_example");
    await user.type(screen.getByLabelText("Root pages"), "https://www.notion.so/Handbook-1f2e");
    await user.click(scan);
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/imports", {
        source: "notion",
        project_id: "p-1",
        from: "https://www.notion.so/Handbook-1f2e",
        credential: "ntn_example",
      }),
    );
  });

  it("offers a Notion scan's docs to import", async () => {
    renderSection([anImport({ source: "notion", total: 8, report: { ...report, tickets: 0, comments: 0, docs: 8, people: [], statuses: [] } })]);
    expect(await screen.findByText("8 docs")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Import 8 docs" })).toBeInTheDocument();
  });

  it("scans a Trello board with its key and token, and drops the form once it starts", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: anImport({ source: "trello", from: "abc", state: "scanning", total: 0 }) });
    renderSection([]);
    await pickOption(user, "Source", "Trello board");
    await pickOption(user, "Project", "Atlas");
    const scan = screen.getByRole("button", { name: "Scan" });
    expect(scan).toBeDisabled();
    await user.type(screen.getByLabelText("Board"), "https://trello.com/b/abc/roadmap");
    const token = screen.getByLabelText("API key and token");
    expect(token).toHaveAttribute("type", "password");
    await user.type(token, "key:token");
    await user.click(scan);
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/imports", {
        source: "trello",
        project_id: "p-1",
        from: "https://trello.com/b/abc/roadmap",
        credential: "key:token",
      }),
    );
    expect(await screen.findByText("Scanning")).toBeInTheDocument();
    expect(screen.queryByLabelText("API key and token")).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/projects/p-1/repos");
  });

  it("asks which repository when the project has several", async () => {
    const user = userEvent.setup();
    renderSection(
      [],
      [
        {
          owner: "acme",
          name: "app",
          full_name: "acme/app",
          connector_id: "github",
          role: "app",
        },
        {
          owner: "acme",
          name: "tests",
          full_name: "acme/tests",
          connector_id: "github",
          role: "tests",
        },
      ],
    );
    await pickOption(user, "Project", "Atlas");
    expect(screen.getByRole("button", { name: "Scan" })).toBeDisabled();
    await pickOption(user, "Repository", "acme/tests");
    expect(screen.getByRole("button", { name: "Scan" })).toBeEnabled();
  });

  it("shows the scan's counts, columns and people, and confirms it", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: anImport({ state: "importing" }) });
    renderSection([anImport()]);

    expect(await screen.findByText("250 tickets · 275 comments")).toBeInTheDocument();
    expect(screen.getByText("· new column")).toBeInTheDocument();
    expect(screen.getByText("Unmatched")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Import 250 tickets" }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/imports/imp-1/confirm"));
    expect(await screen.findByRole("progressbar", { name: "Import progress" })).toBeInTheDocument();
  });

  it("counts what the source keeps archived and leaves out comments it could not count", async () => {
    renderSection([anImport({ source: "trello", report: { ...report, comments: 0, archived: 4 } })]);
    expect(await screen.findByText("250 tickets · 4 archived, not imported")).toBeInTheDocument();
  });

  it("lists an ended import with what it brought in and what failed", async () => {
    renderSection([
      anImport({
        state: "completed",
        done: 249,
        failed: 1,
        report: {
          ...report,
          failures: [
            {
              kind: "ticket",
              source_id: "77",
              title: "Broken one",
              error: "invalid: title refused",
            },
          ],
        },
      }),
    ]);
    expect(await screen.findByText("249 of 250 in")).toBeInTheDocument();
    expect(screen.getByText("Broken one")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Scan" })).toBeInTheDocument();
  });
});

describe("importFollower", () => {
  it("patches the import a progress frame carries into its workspace's list", async () => {
    const client = new QueryClient();
    client.setQueryData([getImportsKey, "ws-1"], [anImport({ state: "importing" }), anImport({ id: "imp-0", state: "completed" })]);
    await followFrame(importFollower, "import.progress", { user_id: "u-1", import: anImport({ state: "importing", done: 120 }) }, client);
    expect(client.getQueryData<Import[]>([getImportsKey, "ws-1"])?.map((i) => [i.id, i.done])).toEqual([
      ["imp-1", 120],
      ["imp-0", 0],
    ]);
    await followFrame(importFollower, "import.progress", { user_id: "u-1", import: anImport({ id: "imp-2", state: "scanning" }) }, client);
    expect(client.getQueryData<Import[]>([getImportsKey, "ws-1"])?.[0]?.id).toBe("imp-2");
  });
});
