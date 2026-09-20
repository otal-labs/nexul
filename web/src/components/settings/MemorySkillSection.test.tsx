import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MemorySkillSection } from "@/components/settings/MemorySkillSection";

describe("MemorySkillSection", () => {
  it("renders the copy-paste skill file with a copy button", () => {
    render(<MemorySkillSection />);

    expect(screen.getByText("nexul-memory")).toBeInTheDocument();
    expect(screen.getByText("SKILL.md", { exact: false })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();

    const snippet = screen.getByText(/name: nexul-memory/).textContent ?? "";
    expect(snippet).toContain("memory_get");
    expect(snippet).toContain("memory_create");
    expect(snippet).toContain("memory_update");
    expect(snippet).toContain("@Agent remember");
  });
});
