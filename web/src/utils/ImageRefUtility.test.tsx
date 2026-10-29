// @vitest-environment node
import { describe, expect, it } from "vitest";

import { elapsedShort, imageShortName, keptTail } from "@/utils/ImageRefUtility";

describe("imageShortName", () => {
  it.each([
    ["ghcr.io/northwind/order-api:2026.10.29-rc.1", "order-api:2026.10.29-rc.1"],
    ["getmeili/meilisearch:v1.9", "meilisearch:v1.9"],
    ["redis", "redis"],
    ["ghcr.io/acme/api@sha256:4b227777d4dd1fc61c6f884f48641d02b4d121d3fd328cb08b5531fcacdabf8a", "api@4b227777d4dd"],
  ])("%s reads as %s", (ref, want) => expect(imageShortName(ref)).toBe(want));
});

describe("keptTail", () => {
  it.each([
    ["order-api:2026.10.29-rc.1", "image", ["order-api", ":2026.10.29-rc.1"]],
    ["api@4b227777d4dd", "image", ["api", "@4b227777d4dd"]],
    ["api", "image", ["api", ""]],
    ["orders.fulfilment.example.com", "hostname", ["orders.fulfilment.", "example.com"]],
    ["example.com", "hostname", ["example.com", ""]],
  ] as const)("%s (%s) splits as %j", (value, kind, want) => expect(keptTail(value, kind)).toEqual(want));
});

describe("elapsedShort", () => {
  const now = Date.parse("2026-10-29T12:00:00Z");
  it.each([
    ["2026-10-29T11:59:40Z", "<1m"],
    ["2026-10-29T11:33:00Z", "27m"],
    ["2026-10-29T10:00:00Z", "2h"],
    ["2026-10-29T09:55:00Z", "2h 5m"],
    ["2026-10-29T12:05:00Z", "<1m"],
  ])("since %s is %s", (from, want) => expect(elapsedShort(from, now)).toBe(want));
});
