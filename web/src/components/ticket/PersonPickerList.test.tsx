import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PersonPickerList } from "@/components/ticket/PersonPickerList";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "" }));

describe("PersonPickerList", () => {
  it("offers the people who may open the ticket's project, leaving out a Restricted member without access", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/projects/p-web/people") return Promise.resolve({ data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice", avatar_url: "" }] } });
      if (url.endsWith("/people")) return Promise.resolve({ data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice", avatar_url: "" }, { user_id: "u-2", login: "bob", display_name: "Bob", avatar_url: "" }] } });
      return Promise.reject(new Error(`unexpected GET ${url}`));
    });
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PersonPickerList projectId="p-web" onSelect={vi.fn()} />
      </QueryClientProvider>,
    );

    expect(await screen.findByRole("button", { name: /Alice/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Bob/ })).not.toBeInTheDocument();
  });

  it("checks whoever holds the role, and drops No one while a search is typed so it never reads as a match", async () => {
    mocks.get.mockResolvedValue({ data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice", avatar_url: "" }] } });
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PersonPickerList projectId="p-web" selected="alice" onSelect={vi.fn()} />
      </QueryClientProvider>,
    );

    expect(await screen.findByRole("button", { name: /Alice/, pressed: true })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /No one/, pressed: false })).toBeInTheDocument();
    await userEvent.type(screen.getByRole("textbox", { name: "Search people" }), "ali");
    expect(screen.queryByRole("button", { name: /No one/ })).not.toBeInTheDocument();
  });
});
