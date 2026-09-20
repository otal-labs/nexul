import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FirstLoginWizardPage } from "@/pages/FirstLoginWizardPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const me = {
  user: {
    id: "u1",
    provider: "github" as const,
    provider_user_id: "42",
    login: "onik97",
    name: "Onik",
    avatar_url: "https://avatar/x",
    can_create_workspace: false,
    first_login_done: false,
    created_at: "2026-08-12T12:00:00Z",
  },
  needs_owner_wizard: false,
  needs_first_login_wizard: true,
};

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FirstLoginWizardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("FirstLoginWizardPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.put.mockReset();
    mocks.errorMessage.mockClear();
  });

  it("shows the provider-sourced profile prefilled in the editable form", async () => {
    mocks.get.mockResolvedValue({ data: me });
    renderPage();
    expect(await screen.findByText("Welcome, Onik")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("Onik")).toBeInTheDocument();
  });

  it("saves the profile then completes the first-login wizard", async () => {
    mocks.get.mockResolvedValue({ data: me });
    mocks.put.mockResolvedValue({ data: { ...me.user, display_name: "Onik" } });
    mocks.post.mockResolvedValue({ data: { ...me, needs_first_login_wizard: false } });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: /continue/i }));
    expect(mocks.put).toHaveBeenCalledWith("/api/auth/profile", {
      display_name: "Onik",
      avatar_override_url: "",
    });
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/onboarding/profile");
  });

  it("shows an error when /me fails", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Profile load failed");
    renderPage();
    expect(await screen.findByText("Profile load failed")).toBeInTheDocument();
  });
});
