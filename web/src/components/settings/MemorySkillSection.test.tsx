import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemorySkillSection } from "@/components/settings/MemorySkillSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: mocks.errorMessage }));

const skill = "---\nname: nexul-memory\n---\n\nCall memory_get, memory_create, memory_update. @Agent remember X.\n";

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemorySkillSection />
    </QueryClientProvider>,
  );
};

describe("MemorySkillSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.errorMessage.mockReset();
  });

  it("renders the skill file the server serves, with a copy button", async () => {
    mocks.get.mockResolvedValue({ data: { skill } });
    renderSection();

    expect(await screen.findByText(/name: nexul-memory/)).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/pairing/memory-skill");
    expect(screen.getByText("SKILL.md", { exact: false })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
  });

  it("shows an error when the skill fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Skill lookup failed");
    renderSection();

    expect(await screen.findByText("Skill lookup failed")).toBeInTheDocument();
  });
});
