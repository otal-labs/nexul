import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupT3CodeStep } from "@/components/auth/SetupT3CodeStep";
import type { Computer } from "@/models/Pairing";

const mocks = vi.hoisted(() => ({ useListComputers: vi.fn() }));

vi.mock("@/hooks/PairingHooks", () => ({ useListComputers: mocks.useListComputers }));
vi.mock("@/components/settings/ComputersSection", () => ({ ComputersSection: () => null }));

const computer = (token_expires_at: string): Computer => ({
  id: "c1",
  name: "alice-laptop",
  server_url: "https://alice-laptop-abcd1234.example.com",
  token_expires_at,
  kind: "t3code",
  harness_version: "",
  created_at: "2026-10-08T00:00:00Z",
  updated_at: "2026-10-08T00:00:00Z",
});

const renderStep = () => render(<SetupT3CodeStep onFinish={vi.fn()} finishing={false} />);

describe("SetupT3CodeStep", () => {
  beforeEach(() => mocks.useListComputers.mockReset());

  it("keeps Finish setup locked with no computer", () => {
    mocks.useListComputers.mockReturnValue({ data: [] });
    renderStep();
    expect(screen.getByRole("button", { name: /finish setup/i })).toBeDisabled();
    expect(screen.getByText("Pair a computer to finish setup.")).toBeInTheDocument();
  });

  it("keeps Finish setup locked while the only computer is still pairing", () => {
    mocks.useListComputers.mockReturnValue({ data: [computer("0001-01-01T00:00:00Z")] });
    renderStep();
    expect(screen.getByRole("button", { name: /finish setup/i })).toBeDisabled();
  });

  it("unlocks Finish setup once a computer is paired", () => {
    mocks.useListComputers.mockReturnValue({ data: [computer("2026-11-07T00:00:00Z")] });
    renderStep();
    expect(screen.getByRole("button", { name: /finish setup/i })).toBeEnabled();
    expect(screen.queryByText("Pair a computer to finish setup.")).not.toBeInTheDocument();
  });
});
