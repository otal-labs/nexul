import { Bug } from "lucide-react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Badge, NoFillBadge } from "@/components/ui/badge";

describe("Badge", () => {
  it("renders children with the default filled variant", () => {
    render(<Badge>Open</Badge>);
    const badge = screen.getByText("Open");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("bg-primary");
  });
});

describe("NoFillBadge", () => {
  it("renders a colored icon and colored text, with no fill/border, when given an icon", () => {
    const { container } = render(
      <NoFillBadge icon={Bug} color="text-red-400">
        bug
      </NoFillBadge>,
    );
    const badge = screen.getByText("bug");
    expect(badge).toHaveClass("text-red-400");
    expect(badge).not.toHaveClass("bg-red-400");
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });

  it("renders a solid-color dot and muted text, with no fill/border, when no icon is given", () => {
    const { container } = render(<NoFillBadge color="bg-cyan-500">frontend</NoFillBadge>);
    const badge = screen.getByText("frontend");
    expect(badge).toHaveClass("text-muted-foreground");
    expect(badge).not.toHaveClass("bg-cyan-500");
    const dot = container.querySelector("[aria-hidden]");
    expect(dot).toHaveClass("bg-cyan-500");
    expect(dot?.tagName).toBe("SPAN");
  });

  it("passes an arbitrary caller-supplied color through unmodified, in icon mode", () => {
    render(
      <NoFillBadge icon={Bug} color="text-violet-400">
        feature
      </NoFillBadge>,
    );
    expect(screen.getByText("feature")).toHaveClass("text-violet-400");
  });
});
