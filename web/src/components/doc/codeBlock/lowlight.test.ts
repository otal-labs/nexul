import { describe, expect, it } from "vitest";

import { codeLanguageOptions } from "@/components/doc/codeBlock/lowlight";

describe("codeLanguageOptions", () => {
  // lowlight's common bundle registers plaintext too; the pinned first entry must not duplicate it (duplicate React keys in the picker).
  it("pins plaintext first without duplicating any value", () => {
    const values = codeLanguageOptions.map((o) => o.value);
    expect(values[0]).toBe("plaintext");
    expect(new Set(values).size).toBe(values.length);
  });
});
