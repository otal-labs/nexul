import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MatchText } from "@/components/MatchText";

const marked = (text: string, match?: string) => render(<MatchText text={text} match={match} />).container.querySelector(".underline")?.textContent;

describe("MatchText", () => {
  it("underlines the first match whatever its case, and nothing when there is no search or no match", () => {
    expect(marked("Nikolai Example", "ai")).toBe("ai");
    expect(marked("Aiko Example", " ai ")).toBe("Ai");
    expect(marked("Aiko Example")).toBeUndefined();
    expect(marked("Aiko Example", "zz")).toBeUndefined();
  });
});
