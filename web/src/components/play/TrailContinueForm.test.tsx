import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TrailContinueForm } from "@/components/play/TrailContinueForm";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() }, errorMessage: vi.fn(() => "Refused") }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }));

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TrailContinueForm trailId="tr-1" />
    </QueryClientProvider>,
  );
};

const trail = (id: string) => ({ data: { id, target_type: "ticket", target_id: "t-1" } });

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  vi.mocked(toast.success).mockReset();
  vi.mocked(toast.info).mockReset();
  vi.mocked(toast.error).mockReset();
});

describe("TrailContinueForm", () => {
  it("sends only the message to the run and clears the box", async () => {
    vi.mocked(api.post).mockResolvedValue(trail("tr-1"));
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Continue this run"), "Keep two threads.");
    await user.click(screen.getByRole("button", { name: "Send" }));

    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/continue", { message: "Keep two threads." });
    expect(await screen.findByLabelText("Continue this run")).toHaveValue("");
    expect(toast.success).toHaveBeenCalledWith("Sent to the run");
  });

  it("says so when the run's thread was gone and the play started again", async () => {
    vi.mocked(api.post).mockResolvedValue(trail("tr-2"));
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Continue this run"), "More.");
    await user.click(screen.getByRole("button", { name: "Send" }));

    await vi.waitFor(() => expect(toast.info).toHaveBeenCalledWith("Its thread was deleted in T3 Code, so the play started again as a new run"));
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("refuses an empty message without calling the server, and shows a refusal", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: "Send" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Write what the agent should do next");
    expect(api.post).not.toHaveBeenCalled();

    vi.mocked(api.post).mockRejectedValue(new Error("409"));
    await user.type(screen.getByLabelText("Continue this run"), "More.");
    await user.click(screen.getByRole("button", { name: "Send" }));
    await vi.waitFor(() => expect(toast.error).toHaveBeenCalledWith("Refused"));
    expect(screen.getByLabelText("Continue this run")).toHaveValue("More.");
  });
});
