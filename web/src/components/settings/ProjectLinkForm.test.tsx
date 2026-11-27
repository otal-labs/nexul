import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import type { Computer, ProjectLink } from "@/models/Pairing";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), del: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const computer = (id: string, name: string, projects: { id: string; repository?: string }[]) =>
  ({
    id,
    name,
    position: 0,
    runs_at_once: 1,
    running: 0,
    chat_project_id: "",
    facts: { t3: { state: "answering" }, providers: [], projects: projects.map((p) => ({ title: p.id, path: `/code/${p.id}`, ...p })) },
  }) as unknown as Computer;

const laptop = computer("c1", "laptop", [{ id: "t3-app", repository: "github.com/acme/app" }]);
const box = computer("c2", "build-box", [
  { id: "t3-app-a", repository: "github.com/acme/app" },
  { id: "t3-app-b", repository: "github.com/acme/app" },
]);
const empty: ProjectLink = { project_id: "proj-1", computers: [], checkouts: [] };

const renderForm = (link = empty) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ProjectLinkForm projectId="proj-1" projectName="Acme App" link={link} />
    </QueryClientProvider>,
  );
};

describe("ProjectLinkForm", () => {
  beforeEach(() => {
    mocks.get.mockReset().mockImplementation(async (url: string) => {
      if (url === "/api/pairing/computers") return { data: { computers: [laptop, box] } };
      if (url === "/api/pairing/defaults") return { data: { placement: "in_order", clone_automatically: false } };
      if (url === "/api/projects/proj-1/repos") return { data: [{ owner: "acme", name: "app", full_name: "acme/app", connector_id: "github", role: "app" }] };
      return { data: {} };
    });
    mocks.put.mockReset().mockImplementation((_url: string, body: object) => Promise.resolve({ data: { project_id: "proj-1", ...body } }));
  });

  it("lists the person's computers in order with the checkout a run would use on each", async () => {
    renderForm();

    const list = await screen.findByRole("list", { name: /this project's computers/i });
    const rows = await within(list).findAllByRole("listitem");
    expect(rows.map((r) => r.textContent)).toEqual([expect.stringContaining("laptop"), expect.stringContaining("build-box")]);
    expect(within(rows[0]!).getByText("t3-app")).toBeInTheDocument();
    expect(within(rows[1]!).getByRole("combobox", { name: /pick the checkout on build-box/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset to defaults" }), "a project on the defaults has nothing to reset").toBeDisabled();
    expect(screen.queryByRole("button", { name: /^save$/i }), "every change writes at once").not.toBeInTheDocument();
  });

  it("saves its own order of computers and a picked checkout", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByText("t3-app");

    await user.click(screen.getByRole("radio", { name: "Only these" }));
    await pickOption(user, /pick the checkout on build-box/i, /t3-app-b/);

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith(
        "/api/pairing/projects/proj-1",
        expect.objectContaining({ computers: ["c1", "c2"], checkouts: [{ computer_id: "c2", repository: "acme/app", harness_project_id: "t3-app-b" }] }),
      ),
    );
  });

  it("shows and saves one checkout per repository when the project has its tests in another", async () => {
    const user = userEvent.setup();
    const both = computer("c3", "desk", [
      { id: "t3-app", repository: "github.com/acme/app" },
      { id: "t3-e2e", repository: "github.com/acme/e2e" },
      { id: "t3-e2e-2", repository: "github.com/acme/e2e" },
    ]);
    const testsOnly = computer("c4", "ci", [{ id: "t3-e2e-ci", repository: "github.com/acme/e2e" }]);
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/pairing/computers") return { data: { computers: [both, testsOnly] } };
      if (url === "/api/pairing/defaults") return { data: { placement: "in_order", clone_automatically: false } };
      if (url === "/api/projects/proj-1/repos")
        return {
          data: [
            { owner: "acme", name: "app", full_name: "acme/app", connector_id: "github", role: "app" },
            { owner: "acme", name: "e2e", full_name: "acme/e2e", connector_id: "github", role: "tests" },
          ],
        };
      return { data: {} };
    });
    renderForm();

    const list = await screen.findByRole("list", { name: /this project's computers/i });
    const [desk, ci] = await within(list).findAllByRole("listitem");
    expect(within(desk!).getByText("t3-app")).toBeInTheDocument();
    expect(within(desk!).getByRole("combobox", { name: "Pick the acme/e2e checkout on desk" })).toBeInTheDocument();
    expect(within(ci!).getByText("No checkout")).toBeInTheDocument();
    expect(within(ci!).getByText("t3-e2e-ci")).toBeInTheDocument();

    await pickOption(user, "Pick the acme/e2e checkout on desk", /t3-e2e-2/);
    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith(
        "/api/pairing/projects/proj-1",
        expect.objectContaining({ checkouts: [{ computer_id: "c3", repository: "acme/e2e", harness_project_id: "t3-e2e-2" }] }),
      ),
    );
  });

  it("takes a computer out of its own order with the X on its row and offers it again from Include a computer", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByText("t3-app");

    await user.click(screen.getByRole("radio", { name: "Only these" }));
    const list = screen.getByRole("list", { name: /this project's computers/i });
    expect(within(list).getAllByRole("button", { name: /^Reorder/ })).toHaveLength(2);
    await user.click(within(list).getByRole("button", { name: "Remove build-box" }));
    expect(within(list).queryByRole("button", { name: /^Remove|^Reorder/ })).not.toBeInTheDocument();
    await user.click(within(list).getByRole("button", { name: "Include a computer" }));
    expect((await screen.findAllByRole("menuitem")).map((i) => i.textContent)).toEqual(["build-box"]);
    await user.keyboard("{Escape}");
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/pairing/projects/proj-1", expect.objectContaining({ computers: ["c1"] })));
  });

  it("leaves where threads start to the defaults until the person picks a place", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByText("t3-app");

    expect(screen.getByRole("combobox", { name: /threads start in/i })).toHaveTextContent("Same as defaults");
    await pickOption(user, /threads start in/i, "New worktree per thread");
    await waitFor(() =>
      expect(mocks.put).toHaveBeenLastCalledWith("/api/pairing/projects/proj-1", expect.objectContaining({ start_in: "worktree", computers: [] })),
    );
  });

  it("goes back to the person's defaults", async () => {
    const user = userEvent.setup();
    mocks.del.mockResolvedValue({});
    renderForm({ project_id: "proj-1", computers: ["c2"], checkouts: [] });

    await user.click(await screen.findByRole("button", { name: "Reset to defaults" }));
    await user.click(within(await screen.findByRole("dialog", { name: /^Reset .* to defaults\?$/ })).getByRole("button", { name: "Reset to defaults" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/projects/proj-1"));
  });

  it("writes a change at once, and puts the saved value back when the write fails", async () => {
    const user = userEvent.setup();
    mocks.put.mockRejectedValueOnce(new Error("refused"));
    renderForm();

    await pickOption(user, "Threads start in", "New worktree per thread");
    await waitFor(() => expect(mocks.put).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Threads start in" })).toHaveTextContent("Same as defaults"));
  });
});
