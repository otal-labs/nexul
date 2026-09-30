import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceGeneralSection } from "@/components/settings/WorkspaceGeneralSection";
import { getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { patch: mocks.patch },
  errorMessage: (error: { response?: { data?: { message?: string } } }) => error.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const acme: Workspace = { id: "ws-1", name: "Acme", slug: "acme", mention_chip_template: "", created_at: "", updated_at: "" };

const Where = () => <p data-testid="location">{useLocation().pathname}</p>;

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getWorkspacesKey], [acme]);
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/acme/configuration/general"]}>
        <Routes>
          <Route
            path="/:workspace/configuration/general"
            element={
              <>
                <WorkspaceGeneralSection workspace={acme} />
                <Where />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return userEvent.setup();
};

const save = () => screen.getByRole("button", { name: "Save" });

describe("WorkspaceGeneralSection", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  });

  it("starts unchanged: Save is off and no warning about links shows", async () => {
    renderSection();
    expect(await screen.findByLabelText("Name")).toHaveValue("Acme");
    expect(screen.getByLabelText("URL")).toHaveValue("acme");
    expect(save()).toBeDisabled();
    expect(screen.queryByText(/old address will stop working/)).not.toBeInTheDocument();
  });

  it("sends only the name when only the name changed, and stays on the same address", async () => {
    mocks.patch.mockResolvedValue({ data: { ...acme, name: "Acme Labs" } });
    const user = renderSection();
    const name = await screen.findByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Acme Labs");
    await user.click(save());

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1", { name: "Acme Labs" });
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/configuration/general");
  });

  it("warns that old links stop working, sends only the slug, and moves this tab onto the new address", async () => {
    mocks.patch.mockResolvedValue({ data: { ...acme, slug: "acme-labs" } });
    const user = renderSection();
    const url = await screen.findByLabelText("URL");
    await user.clear(url);
    await user.type(url, "Acme Labs");

    expect(url).toHaveValue("acme-labs");
    expect(screen.getByText("Links using the old address will stop working.")).toBeInTheDocument();
    await user.click(save());

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1", { slug: "acme-labs" });
    expect(await screen.findByTestId("location")).toHaveTextContent("/acme-labs/configuration/general");
    expect(useWorkspaceStore.getState().selectedWorkspaceSlug).toBe("acme-labs");
  });

  it("refuses a reserved address inline and never sends it", async () => {
    const user = renderSection();
    const url = await screen.findByLabelText("URL");
    await user.clear(url);
    await user.type(url, "settings");

    expect(await screen.findByRole("alert")).toHaveTextContent("That address is used by the app itself");
    expect(save()).toBeDisabled();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("shows the server's refusal of a taken address on the URL field and stays where it is", async () => {
    mocks.patch.mockRejectedValue({ response: { status: 409, data: { message: 'slug "other" is taken by another workspace' } } });
    const user = renderSection();
    const url = await screen.findByLabelText("URL");
    await user.clear(url);
    await user.type(url, "other");
    await user.click(save());

    expect(await screen.findByRole("alert")).toHaveTextContent('slug "other" is taken by another workspace');
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/configuration/general");
  });

  it("fills the URL from the name on Suggest from name", async () => {
    const user = renderSection();
    const name = await screen.findByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Rixwave Labs");
    await user.click(screen.getByRole("button", { name: "Suggest from name" }));

    expect(screen.getByLabelText("URL")).toHaveValue("rixwave-labs");
  });
});
