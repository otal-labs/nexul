import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TrailContinueForm } from "@/components/play/TrailContinueForm";
import type { Trail } from "@/models/Trail";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn() }, errorMessage: vi.fn(() => "Refused") }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }));

const ended: Trail = {
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: "u-alice",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "Be brief.",
  computer_id: "c-1",
  provider: "",
  model: "",
  harness_session_id: "th-1",
  state: "failed",
  started_at: "2026-10-10T10:00:00Z",
  ended_at: "2026-10-10T10:05:00Z",
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
};

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TrailContinueForm trail={ended} />
    </QueryClientProvider>,
  );
};

const trail = (id: string) => ({ data: { id, target_type: "ticket", target_id: "t-1" } });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
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

    await vi.waitFor(() => expect(toast.info).toHaveBeenCalledWith("Its T3 Code thread is gone, so the play started as a new run"));
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

  it("asks where to run when starting the play again needs a location, seeded with the message", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/plays") return { data: [{ id: "play-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket", description: "", builtin_key: "" }] };
      if (url === "/api/plays/latest-choices") return { data: { memory_ids: [], computer_id: "", provider: "", model: "" } };
      if (url === "/api/pairing/resolve") return { data: { ok: true, computer_id: "c-1", harness_project_id: "t3-app" } };
      if (url === "/api/pairing/presence") return { data: { computers: { "c-1": "connected" } } };
      if (url === "/api/pairing/computers") return { data: { computers: [{ id: "c-1", name: "Laptop" }] } };
      if (url === "/api/pairing/projects") return { data: { links: [] } };
      return { data: [] };
    });
    vi.mocked(api.post).mockRejectedValue({ response: { data: { message: "Pick where plays run", details: { reason: "needs_location" } } } });
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText("Continue this run"), "More.");
    await user.click(screen.getByRole("button", { name: "Send" }));

    expect(await screen.findByRole("heading", { name: "Where to run" })).toBeInTheDocument();
    expect(screen.getByLabelText("Instructions for this run")).toHaveValue("Be brief.\n\nMore.");
    expect(toast.error).not.toHaveBeenCalled();
  });
});
