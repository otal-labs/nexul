import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { IntroduceYourselfStep } from "@/components/auth/IntroduceYourselfStep";
import type { User } from "@/models/User";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const baseUser: User = {
  id: "u1",
  provider: "github",
  provider_user_id: "42",
  login: "onik97",
  name: "Onik GitHub",
  avatar_url: "https://avatar/provider.png",
  can_create_workspace: false,
  first_login_done: false,
  created_at: "2026-08-12T12:00:00Z",
};

const renderStep = (user: User, onContinue = vi.fn()) => {
  mocks.get.mockResolvedValue({ data: { user, needs_owner_wizard: false, needs_first_login_wizard: true } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <IntroduceYourselfStep onContinue={onContinue} />
    </QueryClientProvider>,
  );
  return onContinue;
};

// Minimal FileReader stub: jsdom implements readAsDataURL, but real image
// decoding isn't needed since we only check the resulting data URI is submitted.
class FakeFileReader {
  onload: (() => void) | null = null;
  result: string | null = null;
  readAsDataURL(file: File) {
    this.result = `data:${file.type};base64,ZmFrZQ==`;
    this.onload?.();
  }
}

describe("IntroduceYourselfStep", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.errorMessage.mockReset();
    mocks.errorMessage.mockImplementation((e: unknown) => (e as Error)?.message ?? "Something went wrong");
    vi.stubGlobal("FileReader", FakeFileReader);
  });

  it("prefills the name with the effective name and previews the effective avatar", async () => {
    renderStep({ ...baseUser, display_name: "Onik Override", avatar_override_url: "data:image/png;base64,old" });
    expect(await screen.findByDisplayValue("Onik Override")).toBeInTheDocument();
    const img = (await screen.findByAltText("")) as HTMLImageElement;
    expect(img.src).toContain("data:image/png;base64,old");
  });

  it("falls back to the provider-sourced name/avatar when no override is set", async () => {
    renderStep({ ...baseUser });
    expect(await screen.findByDisplayValue("Onik GitHub")).toBeInTheDocument();
    const img = (await screen.findByAltText("")) as HTMLImageElement;
    expect(img.src).toBe(baseUser.avatar_url);
  });

  it("uploads a file, updates the preview, and submits it as avatar_override_url", async () => {
    const onContinue = renderStep({ ...baseUser });
    mocks.put.mockResolvedValue({ data: { ...baseUser, display_name: "Onik GitHub" } });
    const user = userEvent.setup();

    await screen.findByDisplayValue("Onik GitHub");
    const file = new File(["fake-image-bytes"], "avatar.png", { type: "image/png" });
    await user.upload(screen.getByLabelText("Upload avatar image"), file);

    const img = (await screen.findByAltText("")) as HTMLImageElement;
    await waitFor(() => expect(img.src).toContain("data:image/png;base64,ZmFrZQ=="));

    await user.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/auth/profile", {
        display_name: "Onik GitHub",
        avatar_override_url: "data:image/png;base64,ZmFrZQ==",
      }),
    );
    expect(onContinue).toHaveBeenCalled();
  });

  it("clicking Remove clears the avatar override back to provider-sourced on submit", async () => {
    const onContinue = renderStep({
      ...baseUser,
      display_name: "Onik Override",
      avatar_override_url: "data:image/png;base64,old",
    });
    mocks.put.mockResolvedValue({ data: { ...baseUser } });
    const user = userEvent.setup();

    await screen.findByDisplayValue("Onik Override");
    await user.click(screen.getByRole("button", { name: /remove/i }));
    await user.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/auth/profile", {
        display_name: "Onik Override",
        avatar_override_url: "",
      }),
    );
    expect(onContinue).toHaveBeenCalled();
  });

  it("surfaces a backend validation error inline instead of crashing", async () => {
    const onContinue = renderStep({ ...baseUser });
    mocks.put.mockRejectedValue(new Error("avatar image exceeds the 10MB limit"));
    const user = userEvent.setup();

    await screen.findByDisplayValue("Onik GitHub");
    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(await screen.findByText("avatar image exceeds the 10MB limit")).toBeInTheDocument();
    expect(onContinue).not.toHaveBeenCalled();
  });
});
