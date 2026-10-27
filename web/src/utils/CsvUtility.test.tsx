// @vitest-environment node
import { describe, expect, it } from "vitest";

import { csvDelimiter, parseCsv } from "@/utils/CsvUtility";

describe("parseCsv", () => {
  it("keeps delimiters, line breaks and doubled quotes inside a quoted field", () => {
    const table = parseCsv('name,note\r\nalice,"one, two"\r\nbob,"line\nbreak ""quoted"""\r\n', ",", 10, 10);
    expect(table).toEqual({
      rows: [
        ["name", "note"],
        ["alice", "one, two"],
        ["bob", 'line\nbreak "quoted"'],
      ],
      truncated: false,
    });
  });

  it("drops a leading byte order mark and reads a last line with no break", () => {
    expect(parseCsv("﻿a,b\n1,2", ",", 10, 10).rows).toEqual([
      ["a", "b"],
      ["1", "2"],
    ]);
  });

  it("stops at the row limit and says the file was cut", () => {
    const text = Array.from({ length: 1000 }, (_, i) => `${i},x`).join("\n");
    const table = parseCsv(text, ",", 3, 10);
    expect(table.rows).toEqual([
      ["0", "x"],
      ["1", "x"],
      ["2", "x"],
    ]);
    expect(table.truncated).toBe(true);
  });

  it("keeps only the first columns of a wide row and says the file was cut", () => {
    const table = parseCsv("a,b,c,d\n1,2,3,4\n", ",", 10, 2);
    expect(table).toEqual({ rows: [["a", "b"], ["1", "2"]], truncated: true });
  });

  it("returns no rows for an empty file", () => {
    expect(parseCsv("", ",", 10, 10)).toEqual({ rows: [], truncated: false });
  });
});

describe("csvDelimiter", () => {
  it("splits a TSV on tabs and anything else on commas", () => {
    expect(csvDelimiter("export.TSV")).toBe("\t");
    expect(csvDelimiter("export.csv")).toBe(",");
  });
});
