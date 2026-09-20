import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HealthDot } from "@/components/service/HealthDot";
import { DeployStatus } from "@/models/Service";

describe("HealthDot", () => {
  it("pulses only for the healthy status", () => {
    const { container: healthy } = render(<HealthDot status={DeployStatus.Healthy} />);
    expect(healthy.firstChild).toHaveClass("animate-[status-pulse_2.4s_ease-standard_infinite]");

    const { container: running } = render(<HealthDot status={DeployStatus.Running} />);
    expect(running.firstChild).not.toHaveClass("animate-[status-pulse_2.4s_ease-standard_infinite]");

    const { container: failed } = render(<HealthDot status={DeployStatus.Failed} />);
    expect(failed.firstChild).not.toHaveClass("animate-[status-pulse_2.4s_ease-standard_infinite]");
  });

  it("cross-fades status color changes instead of a hard cut", () => {
    const { container } = render(<HealthDot status={DeployStatus.Pending} />);
    expect(container.firstChild).toHaveClass("transition-colors", "duration-150", "ease-standard");
  });

  it("is decorative — the label lives in the adjacent badge, not the dot", () => {
    const { container } = render(<HealthDot status={DeployStatus.Failed} />);
    expect(container.firstChild).toHaveAttribute("aria-hidden");
  });
});
