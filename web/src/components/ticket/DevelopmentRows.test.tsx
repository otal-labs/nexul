import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { BranchRow } from "@/components/ticket/BranchRow";
import { PRRow } from "@/components/ticket/PRRow";
import type { BranchLink, PRLink } from "@/models/Ticket";

const pr = (state: PRLink["state"]): PRLink => ({
  owner: "acme",
  repo: "app",
  number: 42,
  title: "Fix login",
  sha: "abc",
  state,
});

const branch: BranchLink = { owner: "acme", repo: "app", branch: "ticket/1" };

describe("PRRow", () => {
  it("labels each pull request state", () => {
    const { rerender } = render(
      <ul>
        <PRRow pr={pr("open")} />
      </ul>,
    );
    expect(screen.getByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("open")).toBeInTheDocument();

    rerender(
      <ul>
        <PRRow pr={pr("merged")} />
      </ul>,
    );
    expect(screen.getByText("merged")).toBeInTheDocument();

    rerender(
      <ul>
        <PRRow pr={pr("closed")} />
      </ul>,
    );
    expect(screen.getByText("closed")).toBeInTheDocument();
  });

  it("renders the state as a colored icon + text, with no fill/border", () => {
    const { container, rerender } = render(
      <ul>
        <PRRow pr={pr("open")} />
      </ul>,
    );
    let badge = screen.getByText("open");
    expect(badge).toHaveClass("text-info");
    expect(badge).not.toHaveClass("bg-info/15");
    // Two icons: the row's leading GitPullRequestIcon plus the badge's state icon, both decorative.
    expect(container.querySelectorAll("svg").length).toBe(2);
    expect(container.querySelectorAll("svg[aria-hidden]").length).toBe(2);

    rerender(
      <ul>
        <PRRow pr={pr("merged")} />
      </ul>,
    );
    badge = screen.getByText("merged");
    expect(badge).toHaveClass("text-success");

    rerender(
      <ul>
        <PRRow pr={pr("closed")} />
      </ul>,
    );
    badge = screen.getByText("closed");
    expect(badge).toHaveClass("text-muted-foreground");
  });
});

describe("BranchRow", () => {
  it("renders the owner, repo, and branch name", () => {
    render(
      <ul>
        <BranchRow branch={branch} />
      </ul>,
    );
    expect(screen.getByText("acme/app:ticket/1")).toBeInTheDocument();
  });
});
