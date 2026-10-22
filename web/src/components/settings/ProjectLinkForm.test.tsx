import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

    const list = await screen.findByRole("list", { name: /computers this project's runs try/i });
    const rows = await within(list).findAllByRole("listitem");
    expect(rows.map((r) => r.textContent)).toEqual([expect.stringContaining("laptop"), expect.stringContaining("build-box")]);
    expect(within(rows[0]!).getByText("t3-app")).toBeInTheDocument();
    expect(within(rows[1]!).getByRole("combobox", { name: /pick the checkout on build-box/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();
  });

  it("saves its own order of computers and a picked checkout", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByText("t3-app");

    await user.click(screen.getByRole("switch", { name: /its own computers/i }));
    await pickOption(user, /pick the checkout on build-box/i, /t3-app-b/);
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith(
        "/api/pairing/projects/proj-1",
        expect.objectContaining({ computers: ["c1", "c2"], checkouts: [{ computer_id: "c2", harness_project_id: "t3-app-b" }] }),
      ),
    );
  });

  it("leaves where threads start to the defaults until the person picks a place", async () => {
    const user = userEvent.setup();
    renderForm();
    await screen.findByText("t3-app");

    expect(screen.getByRole("combobox", { name: /new threads start in/i })).toHaveTextContent("Same as your defaults");
    await pickOption(user, /new threads start in/i, "New worktree per thread");
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    await waitFor(() =>
      expect(mocks.put).toHaveBeenLastCalledWith("/api/pairing/projects/proj-1", expect.objectContaining({ start_in: "worktree", computers: [] })),
    );
  });

  it("goes back to following the person's computers and defaults", async () => {
    const user = userEvent.setup();
    mocks.del.mockResolvedValue({});
    renderForm({ project_id: "proj-1", computers: ["c2"], checkouts: [] });

    await user.click(await screen.findByRole("button", { name: /follow my computers and defaults/i }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/projects/proj-1"));
  });
});
