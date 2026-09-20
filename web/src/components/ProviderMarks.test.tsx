import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LoginProviderMarks } from "@/components/ProviderMarks";

describe("LoginProviderMarks", () => {
  it("renders the Google/Discord accessible text for an email login", () => {
    render(<LoginProviderMarks login="client@example.com" />);
    expect(screen.getByTitle("Google or Discord email")).toBeInTheDocument();
    expect(screen.getByText("Google or Discord email")).toBeInTheDocument();
  });

  it("renders the GitHub accessible text for a username login", () => {
    render(<LoginProviderMarks login="octocat" />);
    expect(screen.getByTitle("GitHub username")).toBeInTheDocument();
    expect(screen.getByText("GitHub username")).toBeInTheDocument();
  });
});
