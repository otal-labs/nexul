import { describe, expect, it } from "vitest";

import { hasMarkdownTable } from "@/utils/MarkdownTableUtility";

describe("hasMarkdownTable", () => {
  it.each([
    ["a piped table", "| Push | Build |\n|---|---|\n| 1 | 2m |", true],
    ["a table after prose", "Timings:\n\n| Push | Build |\n| --- | --- |\n| 1 | 2m |", true],
    ["left, centre, right alignment", "| a | b | c |\n| :--- | :---: | ---: |", true],
    ["no outer pipes", "a | b\n--- | ---", true],
    ["a single column", "| a |\n| - |", true],
    ["a table inside a quote", "> | a | b |\n> |---|---|", true],
    ["a table after a closed fence", "```\ncode\n```\n| a | b |\n|---|---|", true],
    ["pipes inside a backtick fence", "```\n| a | b |\n|---|---|\n```", false],
    ["pipes inside a tilde fence", "~~~md\n| a | b |\n|---|---|\n~~~", false],
    ["a fence left open", "```\n| a | b |\n|---|---|", false],
    ["a pipe in a sentence", "Run a | b to pipe it.\nThen carry on.", false],
    ["a setext heading under a piped line", "a | b\n---", false],
    ["a thematic break under prose", "Some text\n\n---\n\nMore", false],
    ["a delimiter row with text in it", "| a | b |\n| --- | x |", false],
    ["no table at all", "## Swap timings\n\n- one\n- two", false],
  ])("%s", (_name, markdown, expected) => {
    expect(hasMarkdownTable(markdown)).toBe(expected);
  });
});
