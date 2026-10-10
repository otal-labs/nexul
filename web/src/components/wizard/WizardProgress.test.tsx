import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter, useParams } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardProgress } from "@/components/wizard/WizardProgress";
import type { ProjectSetup } from "@/models/Project";
import type { WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));

const AtStep = () => {
  const { step } = useParams();
  return (
    <>
      <WizardProgress step={step as WizardStepId} />
      <p>at {step}</p>
    </>
  );
};

const renderAt = (step: string, setup?: ProjectSetup) => {
  if (setup) {
    mocks.get.mockResolvedValue({ data: { id: "p-1", name: "Backend", setup } });
    useProjectWizardStore.getState().setProjectId("p-1", "Backend");
  }
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RouterProvider
        router={createMemoryRouter([{ path: "/acme/wizard/project/:step", element: <AtStep /> }], {
          initialEntries: [`/acme/wizard/project/${step}`],
        })}
      />
    </QueryClientProvider>,
  );
};

const items = () => within(screen.getByRole("list", { name: "Project wizard steps" })).getAllByRole("listitem");
const states = () => items().map((item) => item.getAttribute("data-state"));

beforeEach(() => {
  mocks.get.mockReset();
  useProjectWizardStore.getState().reset();
});

describe("WizardProgress", () => {
  it("shows each step as the project records it: done, skipped, the one on screen, or never visited", async () => {
    renderAt("reach", { finished: false, steps: { project: "done", repository: "skipped" } });

    await vi.waitFor(() => expect(states()).toEqual(["done", "skipped", "unvisited", "current", "unvisited", "unvisited"]));
    expect(items()[3]!.querySelector("[aria-current='step']")).not.toBeNull();
  });

  it("shows Done as done once setup was finished", async () => {
    renderAt("project", { finished: true, steps: { project: "done" } });

    await vi.waitFor(() => expect(states().at(-1)).toBe("done"));
  });

  it("opens any step from the row, whatever happened there, ahead of the one on screen included", async () => {
    const user = userEvent.setup();
    renderAt("repository", { finished: false, steps: { project: "done" } });

    await user.click(await screen.findByRole("button", { name: "Deploy branches" }));
    expect(await screen.findByText("at branches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Info, done" }));
    expect(await screen.findByText("at project")).toBeInTheDocument();
  });

  it("shows one counter and the current label under the row for narrow widths, with the per-step labels hidden until wide", () => {
    renderAt("service");

    const compact = screen.getByText("3 / 6").parentElement!;
    expect(screen.getAllByText("3 / 6")).toHaveLength(1);
    expect(compact).toHaveClass("@2xl:hidden");
    expect(within(compact).getByText("Service")).toHaveClass("font-bold");
    for (const item of items()) expect(item.querySelector("span[aria-hidden]:last-child")).toHaveClass("hidden", "@2xl:block");
  });

  it("counts the Environment step once the scan found env keys", () => {
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: ["A"] });
    renderAt("env");

    expect(screen.getByText("4 / 7")).toBeInTheDocument();
  });
});
