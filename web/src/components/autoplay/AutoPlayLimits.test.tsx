import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutoPlayLimits } from "@/components/autoplay/AutoPlayLimits";
import type { Computer } from "@/models/Pairing";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch },
  errorMessage: (e: { message?: string }) => e?.message ?? "Something went wrong",
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const computer = (id: string, name: string, extra: Partial<Computer> = {}) =>
  ({ id, name, position: 0, runs_at_once: 2, running: 0, chat_project_id: "", ...extra }) as Computer;

let computers: Computer[] = [];

beforeEach(() => {
  computers = [
    computer("c-box", "build-box"),
    computer("c-mint", "mint"),
    computer("c-lena", "lena-laptop", { shared: { owner_id: "u-lena", owner_name: "Lena", permissions: ["computers:see"] } }),
  ];
  mocks.get.mockReset().mockImplementation(async (url: string) => {
    if (url.endsWith("/auto-play-limits")) return { data: { daily_cap_per_ticket: 5, runs_at_once_per_person: 1 } };
    return { data: { computers } };
  });
  mocks.patch.mockReset().mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[1], ...body } }));
});

const wrap = (node: ReactNode) =>
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{node}</QueryClientProvider>);

describe("AutoPlayLimits", () => {
  it("sets the runs at once of the computer picked from, and lists none that are shared with the viewer", async () => {
    wrap(<AutoPlayLimits workspaceId="ws-1" ticket canWrite />);

    const mint = await screen.findByRole("radiogroup", { name: "Runs at once on mint" });
    await userEvent.setup().click(within(mint).getByRole("radio", { name: "3" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-mint", { runs_at_once: 3 });
    expect(screen.getAllByRole("radiogroup", { name: /^Runs at once on/ })).toHaveLength(2);
    expect(screen.queryByText(/lena-laptop/)).not.toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Auto plays per ticket a day" })).toHaveTextContent("5");
  });

  it("shows no Limits block on a play without a daily cap for a viewer with no computers", async () => {
    computers = [];
    wrap(<AutoPlayLimits workspaceId="ws-1" ticket={false} canWrite />);

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByText("Limits")).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith(expect.stringContaining("auto-play-limits"));
  });
});
