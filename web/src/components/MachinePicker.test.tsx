import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MachinePicker } from "@/components/MachinePicker";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

const Harness = () => {
  const form = useForm<{ target: string }>({ defaultValues: { target: "" } });
  return <MachinePicker control={form.control} name="target" />;
};

const renderPicker = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness />
    </QueryClientProvider>,
  );
};

describe("MachinePicker", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
  });

  it("offers machines by name, never runners", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: [{ id: "m1", name: "prod-1", stack_root: "/data/nexul", first_seen: "", last_seen: "" }],
    });
    const user = userEvent.setup();
    renderPicker();

    await user.click(await screen.findByRole("combobox", { name: "Machine" }));
    expect(await screen.findByRole("option", { name: "prod-1" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/machines");
  });

  it("says to connect a runner when there are no machines yet", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    renderPicker();

    expect(await screen.findByText(/no machines yet/i)).toBeInTheDocument();
  });
});
