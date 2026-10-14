import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OpenRunnerCodes } from "@/components/runner/OpenRunnerCodes";
import { OpenComputerCodes } from "@/components/settings/OpenComputerCodes";

const mocks = vi.hoisted(() => ({ get: vi.fn(), del: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, delete: mocks.del }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const runnerCode = { id: "c-1", name: "build-box", machine: "prod", created_at: "2026-10-14T10:00:00Z", expires_at: "2026-10-14T11:00:00Z" };
const computerCode = { id: "c-2", name: "computer-ab12cd34", computer_id: "comp-1", created_at: "2026-10-14T10:00:00Z", expires_at: "2026-10-14T11:00:00Z" };

const renderCodes = (ui: React.ReactNode) =>
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>);

beforeEach(() => {
  mocks.get.mockReset();
  mocks.del.mockReset();
});

describe("open enrollment codes", () => {
  it("lists a runner's unused code apart from a computer's and revokes it", async () => {
    const user = userEvent.setup();
    let codes = [runnerCode, computerCode];
    mocks.get.mockImplementation(async () => ({ data: codes }));
    mocks.del.mockImplementation(async () => {
      codes = [computerCode];
    });
    renderCodes(<OpenRunnerCodes />);

    expect(await screen.findByText("build-box")).toBeInTheDocument();
    expect(screen.queryByText("computer-ab12cd34")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Revoke the code for build-box" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/runners/enrollments/c-1");
    await vi.waitFor(() => expect(screen.queryByText("build-box")).not.toBeInTheDocument());
  });

  it("names a computer's unused code by the computer it adds", async () => {
    const user = userEvent.setup();
    mocks.get.mockImplementation(async (url: string) => ({
      data: url === "/api/runners/enrollments" ? [runnerCode, computerCode] : { computers: [{ id: "comp-1", name: "Alice Laptop", token_expires_at: "0001-01-01T00:00:00Z", kind: "t3code", server_url: "" }] },
    }));
    mocks.del.mockResolvedValue({});
    renderCodes(<OpenComputerCodes />);

    await user.click(await screen.findByRole("button", { name: "Revoke the code for Alice Laptop" }));

    expect(mocks.del).toHaveBeenCalledWith("/api/runners/enrollments/c-2");
    expect(screen.queryByText("build-box")).not.toBeInTheDocument();
  });
});
