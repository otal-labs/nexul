// @vitest-environment node
import { describe, expect, it } from "vitest";

import { parseEnv, formatEnv } from "@/lib/env";
import { isUpgradeInProgress } from "@/models/InstanceUpgrade";
import { ServiceFormSchema } from "@/models/Service";
import { cardPerson, LinkPRFormSchema, parseTicketKey, reporterLabel, type Ticket } from "@/models/Ticket";

describe("isUpgradeInProgress", () => {
  it("is false for a null record", () => {
    expect(isUpgradeInProgress(null)).toBe(false);
  });

  it("is true for pending and started records", () => {
    const base = {
      id: "u-1",
      from_version: "v1",
      to_version: "v2",
      error: "",
      requested_by: "user-1",
      created_at: "2026-09-15T00:00:00Z",
      updated_at: "2026-09-15T00:00:00Z",
    };
    expect(isUpgradeInProgress({ ...base, status: "pending" })).toBe(true);
    expect(isUpgradeInProgress({ ...base, status: "started" })).toBe(true);
    expect(isUpgradeInProgress({ ...base, status: "completed" })).toBe(false);
    expect(isUpgradeInProgress({ ...base, status: "failed" })).toBe(false);
  });
});

describe("ticket people", () => {
  it("labels each reporter kind", () => {
    expect(reporterLabel({ kind: "user", login: "onik97" })).toBe("onik97");
    expect(reporterLabel({ kind: "user:mcp", login: "onik97" })).toBe("Nexul · from onik97");
    expect(reporterLabel({ kind: "automation", automation_name: "Triage" })).toBe("Nexul · Triage");
    expect(reporterLabel({ kind: "user:mcp" })).toBe("Nexul");
  });

  it("puts the tester on the card only in a testing stage", () => {
    const t = { developer: "dev", tester: "qa" } as Ticket;
    expect(cardPerson(t, "testing")).toEqual({ role: "tester", login: "qa" });
    expect(cardPerson(t, "review")).toEqual({ role: "developer", login: "dev" });
    expect(cardPerson(t, undefined)).toEqual({ role: "developer", login: "dev" });
  });
});

describe("ServiceFormSchema", () => {
  const base = {
    name: "api",
    target: "h:22",
    compose_dir: "",
    docker_network: "",
    health_url: "http://h/health",
    env: "",
    image: "",
    ref: "",
  };

  it("requires the field its strategy deploys with", () => {
    expect(ServiceFormSchema.safeParse({ ...base, strategy: "compose", compose_dir: "/srv/api" }).success).toBe(true);
    expect(ServiceFormSchema.safeParse({ ...base, strategy: "run", docker_network: "net" }).success).toBe(true);
    expect(ServiceFormSchema.safeParse({ ...base, strategy: "compose" }).success).toBe(false);
    expect(ServiceFormSchema.safeParse({ ...base, strategy: "run" }).success).toBe(false);
  });
});

describe("env blocks", () => {
  it("parses and formats", () => {
    expect(parseEnv("PORT=8080\n# comment\nEMPTY=\nNEXT=2")).toEqual({ PORT: "8080", EMPTY: "", NEXT: "2" });
    expect(formatEnv({ A: "1", B: "2" })).toBe("A=1\nB=2");
  });
});

describe("LinkPRFormSchema", () => {
  it("coerces the number and rejects zero", () => {
    expect(LinkPRFormSchema.parse({ owner: "acme", repo: "app", number: "7" }).number).toBe(7);
    expect(LinkPRFormSchema.safeParse({ owner: "acme", repo: "app", number: 0 }).success).toBe(false);
  });
});

describe("parseTicketKey", () => {
  it.each([
    ["NX-12", { prefix: "NX", number: 12 }],
    ["P1-12", { prefix: "P1", number: 12 }],
    ["v2api-3", { prefix: "v2api", number: 3 }],
  ])("parses %s", (param, want) => {
    expect(parseTicketKey(param)).toEqual(want);
  });

  it.each(["1P-12", "P-12", "PHASE1-12", "P1-", "0d4f6a1e-3c2b-4e8a-9f10-5b7c8d9e0a1b"])("does not parse %s", (param) => {
    expect(parseTicketKey(param)).toBeUndefined();
  });
});
