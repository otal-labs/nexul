import { render, screen } from "@testing-library/react";
import { MemoryRouter, Navigate, Route, Routes } from "react-router";
import { describe, expect, it } from "vitest";

// Router.tsx's buildRoutes() isn't exported (it composes real pages that need heavy mocking to mount), so this
// exercises the exact redirect wiring in isolation the same way OnboardingGate.test.tsx and Layout.test.tsx
// already test their own route slices — the three old onboarding paths must still resolve now that
// onboarding lives under /wizard/onboarding/….
const renderRedirects = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/onboarding/owner" element={<Navigate to="/wizard/onboarding/owner" replace />} />
        <Route path="/onboarding/profile" element={<Navigate to="/wizard/onboarding/profile" replace />} />
        <Route path="/onboarding/dns" element={<Navigate to="/wizard/onboarding/dns" replace />} />
        <Route path="/wizard/onboarding/owner" element={<div>owner-wizard</div>} />
        <Route path="/wizard/onboarding/profile" element={<div>profile-wizard</div>} />
        <Route path="/wizard/onboarding/dns" element={<div>dns-onboarding</div>} />
      </Routes>
    </MemoryRouter>,
  );

describe("onboarding path redirects", () => {
  it("redirects /onboarding/owner to /wizard/onboarding/owner", async () => {
    renderRedirects("/onboarding/owner");
    expect(await screen.findByText("owner-wizard")).toBeInTheDocument();
  });

  it("redirects /onboarding/profile to /wizard/onboarding/profile", async () => {
    renderRedirects("/onboarding/profile");
    expect(await screen.findByText("profile-wizard")).toBeInTheDocument();
  });

  it("redirects /onboarding/dns to /wizard/onboarding/dns", async () => {
    renderRedirects("/onboarding/dns");
    expect(await screen.findByText("dns-onboarding")).toBeInTheDocument();
  });
});
