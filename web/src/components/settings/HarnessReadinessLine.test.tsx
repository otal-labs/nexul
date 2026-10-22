import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import type { HarnessReadiness } from "@/models/Pairing";

const mocks = vi.hoisted(() => ({
  useHarnessReadiness: vi.fn<() => HarnessReadiness | undefined>(),
  useListComputers: vi.fn(),
}));

vi.mock("@/hooks/PairingHooks", () => ({
  useHarnessReadiness: mocks.useHarnessReadiness,
  useListComputers: mocks.useListComputers,
}));

describe("HarnessReadinessLine", () => {
  it("renders nothing while readiness is still loading", () => {
    mocks.useHarnessReadiness.mockReturnValue(undefined);
    mocks.useListComputers.mockReturnValue({ data: undefined });
    render(<HarnessReadinessLine />);
    expect(screen.queryByText(/ready to run plays/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/harness/i)).not.toBeInTheDocument();
  });

  it("names the computer when ready", () => {
    mocks.useHarnessReadiness.mockReturnValue({ state: "ready", computerId: "c1", harnessProjectId: "", provider: "", model: "", modelOptions: [], passedOver: [] });
    mocks.useListComputers.mockReturnValue({ data: [{ id: "c1", name: "Home" }] });
    render(<HarnessReadinessLine />);
    expect(screen.getByText("Ready: @Agent starts on Home")).toBeInTheDocument();
  });

  it("falls back to a plain ready line when the computer name isn't loaded yet", () => {
    mocks.useHarnessReadiness.mockReturnValue({ state: "ready", computerId: "c1", harnessProjectId: "", provider: "", model: "", modelOptions: [], passedOver: [] });
    mocks.useListComputers.mockReturnValue({ data: undefined });
    render(<HarnessReadinessLine />);
    expect(screen.getByText("Ready: @Agent starts on your first computer")).toBeInTheDocument();
  });

  it("shows the reason message when not ready", () => {
    mocks.useHarnessReadiness.mockReturnValue({
      state: "unpaired",
      message: "Add a computer in Settings to run plays.",
    });
    mocks.useListComputers.mockReturnValue({ data: [] });
    render(<HarnessReadinessLine />);
    expect(screen.getByText("Add a computer in Settings to run plays.")).toBeInTheDocument();
  });
});
