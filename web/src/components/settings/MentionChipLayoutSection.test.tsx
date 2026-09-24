import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import type { InstanceSettings } from "@/models/User";

const mocks = vi.hoisted(() => ({ patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { patch: mocks.patch },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const settings: InstanceSettings = {
  instance_url: "https://deploy.example.com",
  settings_version: 1,
  oauth_callback: "https://deploy.example.com/auth/callback",
  mention_chip_template: "{ticket.Ticket} {ticket.Status}",
};

const renderSection = (overrides: Partial<InstanceSettings> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MentionChipLayoutSection settings={{ ...settings, ...overrides }} />
    </QueryClientProvider>,
  );
};

describe("MentionChipLayoutSection", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
  });

  it("seeds the input from the current template and previews it substituted", () => {
    renderSection();
    expect(screen.getByLabelText("Format")).toHaveValue("{ticket.Ticket} {ticket.Status}");
    expect(screen.getByText("Fix login redirect loop In progress")).toBeInTheDocument();
  });

  it("round-trips edits through the input into the live preview", async () => {
    const user = userEvent.setup();
    renderSection();

    const input = screen.getByLabelText("Format");
    await user.clear(input);
    await user.type(input, "{{ticket.Project} {{ticket.Ticket}");

    expect(screen.getByText("ERF-1 Fix login redirect loop")).toBeInTheDocument();
  });

  it("inserts a token at the end of the input via the token button", async () => {
    const user = userEvent.setup();
    renderSection({ mention_chip_template: "" });

    await user.click(screen.getByRole("button", { name: "{ticket.Developer}" }));
    expect(screen.getByLabelText("Format")).toHaveValue("{ticket.Developer}");
  });

  it("saves the template via PATCH on submit", async () => {
    mocks.patch.mockResolvedValue({ data: { mention_chip_template: "{ticket.Status}" } });
    const user = userEvent.setup();
    renderSection();

    const input = screen.getByLabelText("Format");
    await user.clear(input);
    await user.type(input, "{{ticket.Status}");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/auth/settings/mention-chip-template", {
      mention_chip_template: "{ticket.Status}",
    });
  });
});
