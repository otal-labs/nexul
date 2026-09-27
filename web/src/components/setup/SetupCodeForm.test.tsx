import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { SetupCodeForm } from "@/components/setup/SetupCodeForm";
import { useSetupPassStore } from "@/stores/setupPassStore";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

const refuse = (status: number, code: string) => Promise.reject({ response: { status, data: { message: code, code } } });

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  render(
    <QueryClientProvider client={client}>
      <SetupCodeForm />
    </QueryClientProvider>,
  );
  return { invalidate };
};

const submit = async (code: string) => {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/setup code/i), code);
  await user.click(screen.getByRole("button", { name: /continue/i }));
};

describe("SetupCodeForm", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    useSetupPassStore.setState({ token: null, expiresAt: null, code: null });
    window.history.replaceState(null, "", "/setup");
  });

  afterEach(() => window.history.replaceState(null, "", "/"));

  it("says a wrong or expired code is wrong, and stores no pass", async () => {
    mocks.post.mockImplementation(() => refuse(400, "invalid_code"));
    renderForm();
    await submit("nxs_wrong");

    expect(await screen.findByRole("alert")).toHaveTextContent(/wrong or has expired/i);
    expect(useSetupPassStore.getState().token).toBeNull();
  });

  it("says to wait when the server throttles repeated failures", async () => {
    mocks.post.mockImplementation(() => refuse(429, "too_many_requests"));
    renderForm();
    await submit("nxs_wrong");

    expect(await screen.findByRole("alert")).toHaveTextContent(/wait a few minutes/i);
  });

  it("points to sign-in and refreshes the status once setup is already done", async () => {
    mocks.post.mockImplementation(() => refuse(409, "setup_done"));
    const { invalidate } = renderForm();
    await submit("nxs_abc");

    expect(await screen.findByRole("alert")).toHaveTextContent(/sign in instead/i);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["getBootstrapStatus"] });
  });

  it("requires a code before asking the server", async () => {
    renderForm();
    await userEvent.setup().click(screen.getByRole("button", { name: /continue/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/enter the setup code/i);
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("prefills the code from #code= and removes it from the address bar", () => {
    window.history.replaceState(null, "", "/setup#code=nxs_from_link");
    renderForm();

    expect(screen.getByLabelText(/setup code/i)).toHaveValue("nxs_from_link");
    expect(window.location.hash).toBe("");
    expect(window.location.pathname).toBe("/setup");
  });

  it("stores the pass and keeps the code in memory for the handoff link", async () => {
    mocks.post.mockResolvedValue({ data: { token: "pass-1", expires_at: "2999-01-01T00:00:00Z" } });
    renderForm();
    await submit("nxs_abc");

    await vi.waitFor(() => expect(useSetupPassStore.getState().token).toBe("pass-1"));
    expect(mocks.post).toHaveBeenCalledWith("/api/setup/unlock", { code: "nxs_abc" });
    expect(useSetupPassStore.getState().code).toBe("nxs_abc");
  });
});
