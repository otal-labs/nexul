import { normalizeHost, parseConnectLink } from "@/lib/connectLink";

describe("parseConnectLink", () => {
  const cases: { name: string; raw: string; want: { host: string; code: string } | null }[] = [
    { name: "rejects a plain URL", raw: "https://example.com/connect?host=a&code=b", want: null },
    { name: "rejects another path on the scheme", raw: "nexul://open?host=a&code=b", want: null },
    { name: "rejects a link without a code", raw: "nexul://connect?host=https%3A%2F%2Fn.example", want: null },
    { name: "rejects a link without a host", raw: "nexul://connect?code=ABCD-EFGH-JKMN", want: null },
    { name: "rejects an empty scan", raw: "", want: null },
    {
      name: "reads an encoded host and the code",
      raw: "nexul://connect?host=https%3A%2F%2Fnexul.example.com&code=ABCD-EFGH-JKMN",
      want: { host: "https://nexul.example.com", code: "ABCD-EFGH-JKMN" },
    },
    {
      name: "keeps an http host with a port",
      raw: "nexul://connect?host=http%3A%2F%2F10.0.2.2%3A18980&code=ABCD-EFGH-JKMN",
      want: { host: "http://10.0.2.2:18980", code: "ABCD-EFGH-JKMN" },
    },
    {
      name: "tolerates a slash before the query",
      raw: "nexul://connect/?code=ABCD-EFGH-JKMN&host=nexul.example.com",
      want: { host: "https://nexul.example.com", code: "ABCD-EFGH-JKMN" },
    },
  ];

  test.each(cases)("$name", ({ raw, want }) => {
    expect(parseConnectLink(raw)).toEqual(want);
  });
});

describe("normalizeHost", () => {
  test("defaults to https, drops trailing slashes, and keeps an explicit scheme", () => {
    expect(normalizeHost(" nexul.example.com/ ")).toBe("https://nexul.example.com");
    expect(normalizeHost("http://10.0.2.2:18980//")).toBe("http://10.0.2.2:18980");
    expect(normalizeHost("")).toBe("");
  });
});
