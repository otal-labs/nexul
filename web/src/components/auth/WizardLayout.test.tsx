import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WizardLayout } from "@/components/auth/WizardLayout";

describe("WizardLayout", () => {
  it("renders the step indicator, serif title, and muted subtitle", () => {
    render(
      <WizardLayout step={{ current: 1, total: 2 }} title="Set up your instance" subtitle="First step.">
        <p>step content</p>
      </WizardLayout>,
    );
    expect(screen.getByText("1 / 2")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Set up your instance" })).toBeInTheDocument();
    expect(screen.getByText("First step.")).toBeInTheDocument();
    expect(screen.getByText("step content")).toBeInTheDocument();
  });

  it("calls the back action from the ghost back button", async () => {
    const onBack = vi.fn();
    const user = userEvent.setup();
    render(
      <WizardLayout step={{ current: 2, total: 2 }} title="Last step" subtitle="Almost done." onBack={onBack}>
        <p>step content</p>
      </WizardLayout>,
    );
    await user.click(screen.getByRole("button", { name: /back/i }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("renders no back button when no back action is given", () => {
    render(
      <WizardLayout step={{ current: 1, total: 1 }} title="Solo step" subtitle="Only one.">
        <p>step content</p>
      </WizardLayout>,
    );
    expect(screen.queryByRole("button", { name: /back/i })).not.toBeInTheDocument();
    expect(screen.getByText("1 / 1")).toBeInTheDocument();
  });
});
