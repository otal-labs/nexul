import { describe, expect, it } from "vitest";

import { recordRefFromHref } from "@/components/doc/mention/recordLinks";

const origin = "https://nexul.example.com";

describe("recordRefFromHref", () => {
  it.each([
    ["a ticket by key", `${origin}/acme/tickets/ERF-7`, { type: "ticket", id: "ERF-7" }],
    ["a ticket written as a path", "/acme/tickets/0199-ab", { type: "ticket", id: "0199-ab" }],
    ["a doc in a project folder", `${origin}/acme/docs/ERF/d-1`, { type: "doc", id: "d-1" }],
    ["a doc without a folder", `${origin}/acme/docs/d-1`, { type: "doc", id: "d-1" }],
    ["a board", `${origin}/acme/board/p-1`, null],
    ["the docs home", `${origin}/acme/docs`, null],
    ["another workspace", `${origin}/other/tickets/ERF-7`, null],
    ["another host", "https://elsewhere.example.com/acme/tickets/ERF-7", null],
    ["a doc section anchor", `${origin}/acme/docs/d-1#setup`, null],
    ["a canonical mention path", "/tickets/t-1", null],
  ])("%s", (_name, href, want) => {
    expect(recordRefFromHref(href, origin, "acme")).toEqual(want);
  });
});
