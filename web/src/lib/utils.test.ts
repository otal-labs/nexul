import { describe, expect, it } from "vitest";

import { cn } from "@/lib/utils";

describe("cn", () => {
  it.each([
    ["text-micro text-muted-foreground", "text-micro text-muted-foreground"],
    ["text-row text-foreground", "text-row text-foreground"],
    ["text-sm text-row", "text-row"],
    ["text-xs text-initial", "text-initial"],
    ["text-xl text-title", "text-title"],
    ["font-medium font-book", "font-book"],
  ])("keeps a type-scale step beside a colour and replaces another size: %s", (input, merged) => {
    expect(cn(input)).toBe(merged);
  });
});
