// @vitest-environment node
import { describe, expect, it } from "vitest";

import { AutomationKind } from "@/enums/Automation";
import {
  automationNeedsConfiguration,
  configValuesToSave,
  parseConfigSchema,
  parseConfigValues,
  type Automation,
} from "@/models/Automation";

const baseAutomation = (overrides: Partial<Automation> = {}): Automation => ({
  id: "a1",
  workspace_id: "ws-1",
  name: "Ticket finished",
  description: "Moves a ticket to done",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: ["ticket.pr_merged"],
  config_schema: {},
  config_values: {},
  scopes: ["tickets:write"],
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

describe("parseConfigSchema", () => {
  it("returns no fields for an empty or malformed schema", () => {
    expect(parseConfigSchema({})).toEqual({ fields: [] });
    expect(parseConfigSchema(null)).toEqual({ fields: [] });
    expect(parseConfigSchema("not an object")).toEqual({ fields: [] });
    expect(parseConfigSchema({ properties: "not an object" })).toEqual({ fields: [] });
  });

  it("parses the SDK's JSON Schema wire shape", () => {
    const schema = parseConfigSchema({
      type: "object",
      properties: {
        targetStatus: { type: "string", format: "status", title: "Target status", project_id: "p1" },
        notifyChannel: { type: "string", format: "channel", title: "Notify channel" },
        note: { type: "string", default: "hi" },
        broken: "not an object",
      },
      required: ["targetStatus"],
    });
    expect(schema.fields).toHaveLength(3);
    expect(schema.fields[0]).toMatchObject({
      key: "targetStatus",
      label: "Target status",
      type: "status",
      required: true,
      project_id: "p1",
    });
    expect(schema.fields[1]).toMatchObject({ key: "notifyChannel", type: "channel", required: false });
    expect(schema.fields[2]).toMatchObject({ key: "note", type: "string", default: "hi" });
  });

  it("reads boolean, number and integer properties as their own types, defaults as text", () => {
    const schema = parseConfigSchema({
      properties: {
        onlyFailures: { type: "boolean", default: false },
        limit: { type: "number", default: 5 },
        retries: { type: "integer" },
      },
    });
    expect(schema.fields.map((f) => [f.type, f.default])).toEqual([
      ["boolean", "false"],
      ["number", "5"],
      ["number", undefined],
    ]);
  });

  it("falls back to the key as label and string type for unknown formats", () => {
    const schema = parseConfigSchema({ properties: { plain: { type: "string", format: "mystery" } } });
    expect(schema.fields[0]).toMatchObject({ key: "plain", label: "plain", type: "string" });
  });
});

describe("parseConfigValues", () => {
  it("returns an empty record for non-object input", () => {
    expect(parseConfigValues(null)).toEqual({});
    expect(parseConfigValues("nope")).toEqual({});
  });

  it("keeps scalar entries as text and drops the rest", () => {
    expect(parseConfigValues({ a: "1", b: 2, c: false, d: null, e: { nested: true } })).toEqual({
      a: "1",
      b: "2",
      c: "false",
    });
  });
});

describe("configValuesToSave", () => {
  const { fields } = parseConfigSchema({
    properties: {
      note: { type: "string" },
      onlyFailures: { type: "boolean" },
      limit: { type: "number" },
    },
  });

  it("sends booleans and numbers as their JSON types", () => {
    expect(configValuesToSave(fields, { note: "hi", onlyFailures: "true", limit: "12" })).toEqual({
      note: "hi",
      onlyFailures: true,
      limit: 12,
    });
  });

  it("leaves out an empty number and reads a missing boolean as false", () => {
    expect(configValuesToSave(fields, { note: "", limit: " " })).toEqual({ note: "", onlyFailures: false });
  });
});

describe("automationNeedsConfiguration", () => {
  it("is false when there are no required fields", () => {
    const automation = baseAutomation({ config_schema: { properties: {} } });
    expect(automationNeedsConfiguration(automation)).toBe(false);
  });

  it("is true when a required field has no value", () => {
    const automation = baseAutomation({
      config_schema: { properties: { targetStatus: { type: "string", format: "status", title: "Target status" } }, required: ["targetStatus"] },
      config_values: {},
    });
    expect(automationNeedsConfiguration(automation)).toBe(true);
  });

  it("is false once the required field is filled in", () => {
    const automation = baseAutomation({
      config_schema: { properties: { targetStatus: { type: "string", format: "status", title: "Target status" } }, required: ["targetStatus"] },
      config_values: { targetStatus: "status-1" },
    });
    expect(automationNeedsConfiguration(automation)).toBe(false);
  });
});
