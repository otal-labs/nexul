import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter, useParams } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import { WizardProgress } from "@/components/wizard/WizardProgress";
import type { WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const AtStep = () => {
  const { step } = useParams();
  return (
    <>
      <WizardProgress step={step as WizardStepId} />
      <p>at {step}</p>
    </>
  );
};

const renderAt = (step: string) =>
  render(<RouterProvider router={createMemoryRouter([{ path: "/wizard/project/:step", element: <AtStep /> }], { initialEntries: [`/wizard/project/${step}`] })} />);

const items = () => within(screen.getByRole("list", { name: "Project wizard steps" })).getAllByRole("listitem");

beforeEach(() => {
  useProjectWizardStore.getState().reset();
});

describe("WizardProgress", () => {
  it("marks the steps before the current one done, the current one current, and the rest future", () => {
    renderAt("service");

    expect(items().map((item) => item.getAttribute("data-state"))).toEqual([
      "done",
      "done",
      "current",
      "future",
      "future",
      "future",
    ]);
    expect(items()[2]!.querySelector("[aria-current='step']")).not.toBeNull();
    expect(items().filter((item) => item.querySelector("[aria-current]"))).toHaveLength(1);
  });

  it("turns a done step into a control that goes back to it, and never the project step, which already exists", async () => {
    const user = userEvent.setup();
    renderAt("service");

    expect(within(items()[0]!).queryByRole("button")).not.toBeInTheDocument();
    await user.click(within(items()[1]!).getByRole("button", { name: "Back to Repository" }));

    expect(await screen.findByText("at repository")).toBeInTheDocument();
  });

  it("locks every done step once the stack exists, since revisiting Service would create a second one", () => {
    useProjectWizardStore.getState().setStackId("stack-1");
    renderAt("reach");

    expect(items()[1]).toHaveAttribute("data-state", "done");
    expect(screen.queryByRole("button", { name: /^Back to/ })).not.toBeInTheDocument();
  });

  it("disables the future steps so they cannot be jumped to", () => {
    renderAt("service");

    for (const label of ["Reach", "Deploy branches", "Done"]) {
      expect(within(items().find((item) => item.textContent?.includes(label))!).getByRole("button")).toBeDisabled();
    }
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
