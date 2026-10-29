import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HistoryText } from "@/components/history/HistoryText";

describe("HistoryText", () => {
  it("keeps the spaces at the ends of a changed run outside its mark", () => {
    const { container } = render(
      <p>
        <HistoryText
          parts={[
            { type: "context", text: "Release runbook" },
            { type: "add", text: " (platform) " },
          ]}
        />
      </p>,
    );

    expect(screen.getByText("(platform)").tagName).toBe("INS");
    expect(container).toHaveTextContent("Release runbook (platform)");
  });
});
