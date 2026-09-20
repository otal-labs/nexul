import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OnboardingGate } from "@/components/auth/OnboardingGate";
import type { MeResponse } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), errorMessage: vi.fn(() => "Something went wrong") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: mocks.errorMessage,
}));

const user = {
  id: "u1",
  provider: "github" as const,
  provider_user_id: "42",
  login: "onik97",
  name: "Onik",
  avatar_url: "",
  can_create_workspace: false,
  first_login_done: false,
  created_at: "2026-08-12T12:00:00Z",
};

const me = (overrides: Partial<MeResponse>): MeResponse => ({
  user,
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
  ...overrides,
});

const renderGate = (path = "/") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route
            path="/"
            element={
              <OnboardingGate>
                <div>gated-content</div>
              </OnboardingGate>
            }
          />
          <Route path="/wizard/onboarding/owner" element={<div>owner-wizard</div>} />
          <Route path="/wizard/onboarding/profile" element={<div>profile-wizard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("OnboardingGate", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useSessionStore.setState({ token: "t", isLoggedIn: true });
  });

  it("renders children when onboarding is complete", async () => {
    mocks.get.mockResolvedValue({ data: me({}) });
    renderGate();
    expect(await screen.findByText("gated-content")).toBeInTheDocument();
  });

  it("redirects to the owner wizard on a fresh instance", async () => {
    mocks.get.mockResolvedValue({ data: me({ needs_owner_wizard: true }) });
    renderGate();
    expect(await screen.findByText("owner-wizard")).toBeInTheDocument();
  });

  it("redirects to the first-login wizard for a new member", async () => {
    mocks.get.mockResolvedValue({ data: me({ needs_first_login_wizard: true }) });
    renderGate();
    expect(await screen.findByText("profile-wizard")).toBeInTheDocument();
  });

  it("renders an error display when /me fails", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    renderGate();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });
});
