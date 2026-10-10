import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { GitHubAppForm } from "@/components/setup/GitHubAppForm";

const mocks = vi.hoisted(() => ({ post: vi.fn(), assign: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { post: mocks.post }, errorMessage: () => "Registration failed, start again", joinAPIURL: (path: string) => path }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

beforeEach(() => {
  mocks.post.mockReset();
  mocks.assign.mockReset();
  window.history.replaceState(null, "", "/setup");
  vi.stubGlobal("location", { ...window.location, search: "", assign: mocks.assign });
});
afterEach(() => vi.unstubAllGlobals());

const show = () => render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><GitHubAppForm instanceUrl="https://nexul.example.com" /></QueryClientProvider>);

it("starts an authorized registration and posts the server's public manifest to GitHub", async () => {
  const submit = vi.spyOn(HTMLFormElement.prototype, "submit").mockImplementation(() => {});
  const manifest = { url: "https://nexul.example.com", redirect_url: "https://nexul.example.com/auth/github/manifest/callback" };
  mocks.post.mockResolvedValue({ data: { url: "https://github.com/settings/apps/new?state=signed-state", manifest } });
  show();
  await userEvent.setup().click(screen.getByRole("button", { name: "Create App on GitHub" }));
  expect(mocks.post).toHaveBeenCalledWith("/api/setup/github-app/start");
  await waitFor(() => expect(submit).toHaveBeenCalledOnce());
  const form = submit.mock.instances[0];
  expect(form).toHaveAttribute("action", "https://github.com/settings/apps/new?state=signed-state");
  expect(form).toHaveAttribute("method", "post");
  if (!(form instanceof HTMLFormElement)) throw new Error("Expected GitHub registration form");
  expect(form.querySelector<HTMLInputElement>('input[name="manifest"]')?.value).toBe(JSON.stringify(manifest));
  submit.mockRestore();
});

it("removes the registration code from the URL and finishes setup without exposing the returned credentials", async () => {
  vi.stubGlobal("location", { ...window.location, hash: "#github_manifest=1&code=conversion-code&state=signed-state", assign: mocks.assign });
  const replace = vi.spyOn(window.history, "replaceState");
  mocks.post.mockResolvedValue({ data: undefined });
  show();
  expect(replace).toHaveBeenCalledWith(null, "", "/setup");
  await userEvent.setup().click(screen.getByRole("button", { name: "Finish setup" }));
  expect(mocks.post).toHaveBeenCalledWith("/api/setup/github-app/callback", { code: "conversion-code", state: "signed-state" });
  await waitFor(() => expect(mocks.assign).toHaveBeenCalledWith("/auth/github"));
  expect(screen.queryByLabelText("Private key")).not.toBeInTheDocument();
  replace.mockRestore();
});

it("keeps a failed conversion visible and offers a new registration instead of navigating to sign-in", async () => {
  vi.stubGlobal("location", { ...window.location, hash: "#github_manifest=1&code=conversion-code&state=signed-state", assign: mocks.assign });
  mocks.post.mockRejectedValue(new Error("rejected"));
  show();
  await userEvent.setup().click(screen.getByRole("button", { name: "Finish setup" }));
  expect(await screen.findByText("Registration failed, start again")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Create App on GitHub" })).toBeEnabled();
  expect(mocks.assign).not.toHaveBeenCalled();
});
