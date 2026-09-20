import { describe, expect, it } from "vitest";
import { defaultConfigValues, toJsonSchema, type ConfigSchema } from "../src/config-schema.ts";

const schema = {
  channel: { type: "string", label: "Chat channel", enum: ["general", "eng"] },
  threshold: { type: "number", default: 5 },
  enabled: { type: "boolean" },
} satisfies ConfigSchema;

describe("defaultConfigValues", () => {
  it("uses the declared default when present", () => {
    expect(defaultConfigValues(schema).threshold).toBe(5);
  });

  it("falls back to the type's zero value otherwise", () => {
    const values = defaultConfigValues(schema);
    expect(values.channel).toBe("");
    expect(values.enabled).toBe(false);
  });
});

describe("toJsonSchema", () => {
  it("renders each field as a JSON Schema property, carrying label/default/enum through", () => {
    const jsonSchema = toJsonSchema(schema) as { properties: Record<string, unknown> };
    expect(jsonSchema.properties.channel).toEqual({ type: "string", title: "Chat channel", enum: ["general", "eng"] });
    expect(jsonSchema.properties.threshold).toEqual({ type: "number", default: 5 });
    expect(jsonSchema.properties.enabled).toEqual({ type: "boolean" });
  });
});
