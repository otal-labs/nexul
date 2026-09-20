import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Container } from "@/components/Container";

describe("Container", () => {
  it("renders children", () => {
    render(
      <Container>
        <span>content</span>
      </Container>,
    );
    expect(screen.getByText("content")).toBeInTheDocument();
  });

  it("merges a className prop", () => {
    const { container } = render(<Container className="pt-4" />);
    expect(container.firstChild).toHaveClass("pt-4");
    expect(container.firstChild).toHaveClass("max-w-7xl");
  });
});
