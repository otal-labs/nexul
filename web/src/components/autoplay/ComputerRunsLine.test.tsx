import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputerRunsLine } from "@/components/autoplay/ComputerRunsLine";
import type { Computer } from "@/models/Pairing";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch },
  errorMessage: (e: { message?: string }) => e?.message ?? "Something went wrong",
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const computer = (id: string, name: string, extra: Partial<Computer> = {}) =>
  ({ id, name, position: 0, runs_at_once: 2, running: 0, chat_project_id: "", ...extra }) as Computer;

const computers = [
  computer("c-box", "build-box"),
  computer("c-mint", "mint"),
  computer("c-lena", "lena-laptop", { shared: { owner_id: "u-lena", owner_name: "Lena", permissions: ["computers:see"] } }),
];

beforeEach(() => {
  mocks.get.mockReset().mockResolvedValue({ data: { computers } });
  mocks.patch.mockReset().mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[1], ...body } }));
});

describe("ComputerRunsLine", () => {
  it("sets the runs at once of the computer picked from, and lists none that are shared with the viewer", async () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <ComputerRunsLine />
      </QueryClientProvider>,
    );

    await pickOption(userEvent.setup(), "Runs at once on mint", "3");

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-mint", { runs_at_once: 3 });
    expect(screen.getAllByRole("combobox")).toHaveLength(2);
    expect(screen.queryByText(/lena-laptop/)).not.toBeInTheDocument();
  });
});
