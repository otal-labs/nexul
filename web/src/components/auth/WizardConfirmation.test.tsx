import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WizardConfirmation } from "@/components/auth/WizardConfirmation";

describe("WizardConfirmation", () => {
  it("renders the success check with the serif headline and subtitle", () => {
    render(<WizardConfirmation title="Instance registered" subtitle="One more step." />);
    expect(screen.getByRole("heading", { name: "Instance registered" })).toBeInTheDocument();
    expect(screen.getByText("One more step.")).toBeInTheDocument();
  });

  it("renders without a subtitle when none is given", () => {
    render(<WizardConfirmation title="All set" />);
    expect(screen.getByRole("heading", { name: "All set" })).toBeInTheDocument();
    expect(screen.queryByRole("paragraph")).not.toBeInTheDocument();
  });
});
