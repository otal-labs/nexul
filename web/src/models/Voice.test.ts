import { describe, expect, it } from "vitest";

import { roomNames } from "@/models/Voice";

describe("roomNames", () => {
  it.each([
    [["Lena Kowalski"], "Lena"],
    [["Lena Kowalski", "Sam Okafor"], "Lena and Sam"],
    [["Lena Kowalski", "Sam Okafor", "Bob Tanaka"], "Lena, Sam and Bob"],
    [["Lena Kowalski", "Sam Okafor", "Bob Tanaka", "Jo", "Priya Raman"], "Lena, Sam and 3 others"],
  ])("names %j as %s", (names, expected) => {
    expect(roomNames(names)).toBe(expected);
  });
});
