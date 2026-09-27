import { describe, expect, it } from "vitest";

import {
  OwnHttpsFormSchema,
  ProxyDomainFormSchema,
  codeFromFragment,
  handoffLink,
  resolvesHere,
  setupStage,
} from "@/models/Setup";

const IP_ORIGIN = "http://203.0.113.10:5123";

describe("setupStage", () => {
  it.each([
    ["a user exists: setup is closed", { configured: true, setup_open: false }, true, IP_ORIGIN, "done"],
    ["no pass: the code screen first", { configured: false, setup_open: true }, false, IP_ORIGIN, "code"],
    [
      "no pass even with a stored URL: the code screen",
      { configured: false, setup_open: true, instance_url: "https://deploy.example.com" },
      false,
      IP_ORIGIN,
      "code",
    ],
    ["pass, no URL, server install: the domain step", { configured: false, setup_open: true }, true, IP_ORIGIN, "domain"],
    [
      "pass, no URL, desktop install: localhost, no domain step",
      { configured: false, setup_open: true, local: true },
      true,
      "http://localhost:5123",
      "local",
    ],
    [
      "URL stored, still on IP:port: hand off to the domain",
      { configured: false, setup_open: true, instance_url: "https://deploy.example.com" },
      true,
      IP_ORIGIN,
      "handoff",
    ],
    [
      "URL stored, on the domain: the GitHub step",
      { configured: false, setup_open: true, instance_url: "https://deploy.example.com/" },
      true,
      "https://deploy.example.com",
      "github",
    ],
    [
      "an unparseable stored URL never counts as this origin",
      { configured: false, setup_open: true, instance_url: "not a url" },
      true,
      IP_ORIGIN,
      "handoff",
    ],
    [
      "an older server without setup_open still reaches the GitHub step on its origin",
      { configured: true, instance_url: "https://deploy.example.com" },
      true,
      "https://deploy.example.com",
      "github",
    ],
  ])("%s", (_name, status, hasPass, origin, expected) => {
    expect(setupStage(status, hasPass, origin)).toBe(expected);
  });
});

describe("handoffLink", () => {
  it("carries the code in the fragment so it never reaches a server log", () => {
    expect(handoffLink("https://deploy.example.com/", "nxs_a+b")).toBe("https://deploy.example.com/setup#code=nxs_a%2Bb");
  });

  it("links without a fragment when the code is not known", () => {
    expect(handoffLink("https://deploy.example.com", null)).toBe("https://deploy.example.com/setup");
  });
});

describe("codeFromFragment", () => {
  it.each([
    ["#code=nxs_abc", "nxs_abc"],
    ["#code=nxs_a%2Bb", "nxs_a+b"],
    ["#other=1", ""],
    ["", ""],
  ])("%s", (hash, expected) => {
    expect(codeFromFragment(hash)).toBe(expected);
  });
});

describe("resolvesHere", () => {
  const address = { ipv4: "203.0.113.10", ipv6: "2001:db8::1" };

  it.each([
    ["not resolving yet", [], false],
    ["unknown", undefined, false],
    ["somewhere else", ["198.51.100.7"], false],
    ["here and somewhere else", ["203.0.113.10", "198.51.100.7"], false],
    ["only here over IPv4", ["203.0.113.10"], true],
    ["here over both families", ["203.0.113.10", "2001:db8::1"], true],
  ])("%s", (_name, addresses, expected) => {
    expect(resolvesHere(addresses, address)).toBe(expected);
  });
});

describe("OwnHttpsFormSchema", () => {
  it("refuses an http:// address with the reason", () => {
    const result = OwnHttpsFormSchema.safeParse({ url: "http://deploy.example.com" });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toMatch(/https/i);
  });

  it("refuses something that is not an address", () => {
    expect(OwnHttpsFormSchema.safeParse({ url: "deploy" }).success).toBe(false);
  });

  it("accepts an https address", () => {
    expect(OwnHttpsFormSchema.safeParse({ url: " https://deploy.example.com " }).data?.url).toBe(
      "https://deploy.example.com",
    );
  });
});

describe("ProxyDomainFormSchema", () => {
  it.each([
    ["a URL instead of a name", "https://deploy.example.com", false],
    ["a bare word", "deploy", false],
    ["a domain", "Deploy.Example.com", true],
  ])("%s", (_name, domain, ok) => {
    expect(ProxyDomainFormSchema.safeParse({ domain, email: "" }).success).toBe(ok);
  });

  it("allows an empty email but not a malformed one", () => {
    expect(ProxyDomainFormSchema.safeParse({ domain: "a.example.com", email: "nope" }).success).toBe(false);
    expect(ProxyDomainFormSchema.safeParse({ domain: "a.example.com", email: "ops@example.com" }).success).toBe(true);
  });
});
