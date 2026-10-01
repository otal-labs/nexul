import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectDangerZoneSection } from "@/components/settings/ProjectDangerZoneSection";
import type { Project } from "@/models/Project";

const mocks = vi.hoisted(() => ({ get: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-web", name: "Web", prefix: "WEB" } as Project;

const renderSection = (impact: unknown) => {
  mocks.get.mockResolvedValue({ data: impact });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ProjectDangerZoneSection project={project} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ProjectDangerZoneSection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("names the Restricted members who lose access in the delete confirmation", async () => {
    const user = userEvent.setup();
    renderSection({ tickets: 0, repos: 0, services: 0, restricted_members: [{ user_id: "u-1", name: "Fahad" }, { user_id: "u-2", name: "Sam" }] });

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    await user.click(screen.getByRole("button", { name: "Remove project" }));

    expect(await screen.findByText("Fahad and 1 other restricted member lose access.")).toBeInTheDocument();
    expect(screen.getByText("Fahad, Sam")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("adds no line when nobody restricted holds the project", async () => {
    const user = userEvent.setup();
    renderSection({ tickets: 0, repos: 0, services: 0, restricted_members: [] });

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    await user.click(screen.getByRole("button", { name: "Remove project" }));

    expect(await screen.findByText("This project is empty and can be removed. This cannot be undone.")).toBeInTheDocument();
    expect(screen.queryByText(/lose access/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });
});
