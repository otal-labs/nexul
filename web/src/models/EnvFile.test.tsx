import { describe, expect, it } from "vitest";

import { formatEnvFile, parseEnvFile } from "@/models/EnvFile";

describe("parseEnvFile", () => {
  it.each([
    ["plain pair", "A=1", { A: "1" }],
    ["blank lines and comments are ignored", "\n# note\nA=1\n   \n  # indented\n", { A: "1" }],
    ["export prefix is dropped", "export A=1", { A: "1" }],
    ["double quotes are stripped", 'A="hello world"', { A: "hello world" }],
    ["single quotes are stripped", "A='hello world'", { A: "hello world" }],
    ["escapes inside double quotes stay literal", 'A="line\\nbreak"', { A: "line\\nbreak" }],
    ["only the first = splits", "A=b=c==", { A: "b=c==" }],
    ["whitespace around key and unquoted value is trimmed", "  A  =  1 2  ", { A: "1 2" }],
    ["quoted value keeps its inner spaces", 'A=" x "', { A: " x " }],
    ["a lone quote is a literal value", 'A="', { A: '"' }],
    ["empty value", "A=", { A: "" }],
    ["CRLF line endings", "A=1\r\nB=2\r\n", { A: "1", B: "2" }],
  ])("%s", (_name, text, values) => {
    expect(parseEnvFile(text)).toEqual({ values, invalidLines: [], duplicateKeys: [] });
  });

  it("reports the 1-based line of anything that is not KEY=value, counting blanks and comments", () => {
    const parsed = parseEnvFile("# head\nA=1\n\nnot a pair\n=novalue\n9BAD=1\nBAD-KEY=1\nexport ONLYKEY\nB=2");

    expect(parsed.invalidLines).toEqual([4, 5, 6, 7, 8]);
    expect(parsed.values).toEqual({ A: "1", B: "2" });
  });

  it("lets the last duplicate win and names the key once", () => {
    const parsed = parseEnvFile("A=1\nA=2\nA=3");

    expect(parsed.values).toEqual({ A: "3" });
    expect(parsed.duplicateKeys).toEqual(["A"]);
  });
});

describe("formatEnvFile", () => {
  it("writes the keys in the given order, empty for unset ones, quoting only what a reader would trim or unquote", () => {
    expect(formatEnvFile({ B: " padded ", A: "plain", D: '"q"' }, ["A", "B", "C", "D"])).toBe(
      'A=plain\nB=" padded "\nC=\nD=\'"q"\'',
    );
  });
});
