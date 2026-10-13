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

const person = (login: string) => ({ user_id: login, login, display_name: "", avatar_url: "" });

const renderSection = (impact: unknown, people: unknown[]) => {
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url.endsWith("/people") ? { people, role_ids: [] } : impact }));
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

  it("says how many people lose the project in the delete confirmation", async () => {
    const user = userEvent.setup();
    renderSection({ tickets: 0, repos: 0, services: 0 }, [person("bob"), person("sam")]);

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
    await user.click(screen.getByRole("button", { name: "Remove project" }));

    expect(await screen.findByText("2 people lose this project.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("adds no line when nobody is in the project", async () => {
    const user = userEvent.setup();
    renderSection({ tickets: 0, repos: 0, services: 0 }, []);

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
    await user.click(screen.getByRole("button", { name: "Remove project" }));

    expect(await screen.findByText("The project is empty. Removing it can't be undone.")).toBeInTheDocument();
    expect(screen.queryByText(/lose this project/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });
});
