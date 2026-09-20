import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { RunnerPicker } from "@/components/RunnerPicker";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

const Harness = () => {
  const form = useForm<{ target: string }>({ defaultValues: { target: "" } });
  return <RunnerPicker control={form.control} name="target" />;
};

const renderPicker = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Harness />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("RunnerPicker", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
  });

  it("renders runner options from useRunners, flagging offline ones", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: [
        { id: "r1", name: "instance", connected: true, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
        { id: "r2", name: "spare", connected: false, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
      ],
    });
    const user = userEvent.setup();
    renderPicker();

    await user.click(await screen.findByRole("combobox", { name: "Runs on" }));
    expect(await screen.findByRole("option", { name: "instance" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "spare (offline)" })).toBeInTheDocument();
  });

  it("shows a link to the Runners page when there are no runners yet", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    renderPicker();

    expect(await screen.findByText(/no runners yet/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /add one from the runners page/i })).toHaveAttribute("href", "/runners");
  });
});
