import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AllowlistSection } from "@/components/settings/AllowlistSection";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const membersResponse = { members: ["octocat", "client@example.com"] };

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AllowlistSection />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/members/lookup") return { data: { matches: [] } };
    return { data: membersResponse };
  });
});

describe("AllowlistSection", () => {
  it("shows the right provider mark for a GitHub login and an email login", async () => {
    renderSection();
    await screen.findByText("octocat");
    expect(screen.getByTitle("GitHub username")).toBeInTheDocument();
    expect(screen.getByTitle("Google or Discord email")).toBeInTheDocument();
  });

  it("calls the lookup endpoint once after the debounce and shows the matches", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/members/lookup") {
        return { data: { matches: [{ login: "octocat", avatar_url: "https://avatar/octocat" }] } };
      }
      return { data: membersResponse };
    });
    const user = userEvent.setup();
    renderSection();
    const input = await screen.findByRole("textbox", { name: /github username or email/i });

    await user.type(input, "oct");

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/members/lookup", expect.objectContaining({ params: { q: "oct" } })));
    expect(api.get).toHaveBeenCalledTimes(2); // members list + one debounced lookup call
    expect(await screen.findByRole("listbox", { name: /github username suggestions/i })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /octocat/i })).toBeInTheDocument();
  });

  it("clicking a match fills the input and hides the suggestion list", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/members/lookup") {
        return { data: { matches: [{ login: "octocat", avatar_url: "https://avatar/octocat" }] } };
      }
      return { data: membersResponse };
    });
    const user = userEvent.setup();
    renderSection();
    const input = await screen.findByRole<HTMLInputElement>("textbox", { name: /github username or email/i });

    await user.type(input, "oct");
    await screen.findByRole("option", { name: /octocat/i });
    await user.click(screen.getByRole("option", { name: /octocat/i }));

    expect(input).toHaveValue("octocat");
    expect(screen.queryByRole("listbox", { name: /github username suggestions/i })).not.toBeInTheDocument();
  });

  it("never calls the lookup endpoint while typing an email", async () => {
    const user = userEvent.setup();
    renderSection();
    const input = await screen.findByRole("textbox", { name: /github username or email/i });

    await user.type(input, "client@example.com");
    await new Promise((resolve) => setTimeout(resolve, 400));

    expect(api.get).not.toHaveBeenCalledWith("/api/auth/members/lookup", expect.anything());
  });
});
