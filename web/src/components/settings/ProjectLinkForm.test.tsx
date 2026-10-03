import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
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

const home = { id: "c1", name: "home" } as Computer;
const link: ProjectLink = { project_id: "proj-1", computer_id: "c1", harness_project_id: "t3-proj" };

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ProjectLinkForm projectId="proj-1" projectName="Nexul" link={link} computers={[home]} />
    </QueryClientProvider>,
  );
};

describe("ProjectLinkForm", () => {
  beforeEach(() => {
    mocks.get.mockReset().mockResolvedValue({ data: {} });
    mocks.put.mockReset().mockImplementation((_url: string, body: unknown) => Promise.resolve({ data: body }));
  });

  it("leaves where threads start to the defaults until the person picks a place", async () => {
    const user = userEvent.setup();
    renderForm();

    expect(screen.getByRole("combobox", { name: /new threads start in/i })).toHaveTextContent("Same as my defaults");
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/pairing/projects/proj-1", expect.objectContaining({ start_in: "" })));

    await pickOption(user, /new threads start in/i, "New worktree per thread");
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    await waitFor(() =>
      expect(mocks.put).toHaveBeenLastCalledWith("/api/pairing/projects/proj-1", expect.objectContaining({ start_in: "worktree" })),
    );
  });
});
