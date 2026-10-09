// Reads the published event contract (internal/eventcatalog/schemas.json, generated from the server's payload types)
// and emits typed payloads + fixtures for the SDK. The schemas only use
// object/string/number/integer/boolean/array/null/enum/additionalProperties, so a small recursive mapper covers
// them without a dependency.
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const here = path.dirname(fileURLToPath(import.meta.url));
export const CATALOG_PATH = path.resolve(here, "../../internal/eventcatalog/schemas.json");
export const GENERATED_TS_PATH = path.resolve(here, "../src/events.generated.ts");

export interface JsonSchema {
  type?: string | string[];
  properties?: Record<string, JsonSchema>;
  required?: string[];
  items?: JsonSchema;
  enum?: string[];
  additionalProperties?: JsonSchema;
  [key: string]: unknown;
}

// readCatalog parses the contract file: one JSON object mapping each topic to its schema.
export function readCatalog(json: string): Map<string, JsonSchema> {
  return new Map(Object.entries(JSON.parse(json) as Record<string, JsonSchema>));
}

function schemaType(schema: JsonSchema): string[] {
  if (Array.isArray(schema.type)) return schema.type;
  if (schema.type) return [schema.type];
  return [];
}

function toTsType(schema: JsonSchema): string {
  const types = schemaType(schema);
  if (types.length > 1) {
    return types.map((t) => toTsType({ ...schema, type: t })).join(" | ");
  }
  const type = types[0];
  if (type === "object") {
    if (schema.properties) return toTsObject(schema);
    if (schema.additionalProperties) return `Record<string, ${toTsType(schema.additionalProperties)}>`;
    return "Record<string, unknown>";
  }
  if (type === "array") {
    if (!schema.items) return "unknown[]";
    const item = toTsType(schema.items);
    return item.includes(" | ") ? `(${item})[]` : `${item}[]`;
  }
  if (type === "string") return schema.enum ? schema.enum.map((v) => JSON.stringify(v)).join(" | ") : "string";
  if (type === "integer" || type === "number") return "number";
  if (type === "boolean") return "boolean";
  if (type === "null") return "null";
  return "unknown";
}

function toTsObject(schema: JsonSchema): string {
  const required = new Set(schema.required ?? []);
  const props = Object.entries(schema.properties ?? {});
  if (props.length === 0) return "Record<string, unknown>";
  const fields = props.map(([name, propSchema]) => {
    const optional = required.has(name) ? "" : "?";
    return `${JSON.stringify(name)}${optional}: ${toTsType(propSchema)};`;
  });
  return `{ ${fields.join(" ")} }`;
}

// toFixture builds one representative value satisfying schema — used both as
// the SDK's shipped fixture events and as `dev`'s "fire this event" payload.
// Every optional field is filled in too: a fixture that only had required
// fields would under-exercise handler code that reads the optional ones.
function toFixture(schema: JsonSchema, fieldName = "value"): unknown {
  const types = schemaType(schema);
  const type = types.find((t) => t !== "null") ?? types[0];
  if (type === "object" || (!type && schema.properties)) {
    if (schema.properties) {
      const out: Record<string, unknown> = {};
      for (const [name, propSchema] of Object.entries(schema.properties)) {
        out[name] = toFixture(propSchema, name);
      }
      return out;
    }
    return {};
  }
  if (type === "array") return schema.items ? [toFixture(schema.items, fieldName)] : [];
  if (type === "string" || (!type && schema.enum)) {
    if (schema.enum?.length) return schema.enum[0];
    if (schema.format === "date-time") return "2026-01-01T00:00:00Z";
    return `fixture-${fieldName}`;
  }
  if (type === "integer" || type === "number") return 1;
  if (type === "boolean") return false;
  return null;
}

const GENERATED_HEADER = `// GENERATED FILE — do not edit by hand.
// Source of truth: internal/eventcatalog/schemas.json (make event-schemas).
// Regenerate: bun run generate:events (from sdk/).
`;

export function generateSource(catalog: Map<string, JsonSchema>): string {
  const topics = [...catalog.keys()].sort();
  const payloadFields = topics.map((topic) => `  ${JSON.stringify(topic)}: ${toTsObject(catalog.get(topic)!)};`);
  const fixtureFields = topics.map((topic) => `  ${JSON.stringify(topic)}: ${JSON.stringify(toFixture(catalog.get(topic)!))},`);
  return `${GENERATED_HEADER}
export interface EventPayloads {
${payloadFields.join("\n")}
}

export type Topic = keyof EventPayloads;

export const TOPICS: Topic[] = [
${topics.map((t) => `  ${JSON.stringify(t)},`).join("\n")}
];

export const eventFixtures: { [K in Topic]: EventPayloads[K] } = ${fixtureFields.length ? `{\n${fixtureFields.join("\n")}\n}` : "{}"};
`;
}

function main() {
  const catalog = readCatalog(readFileSync(CATALOG_PATH, "utf8"));
  if (catalog.size === 0) throw new Error(`no event schemas found in ${CATALOG_PATH}`);
  writeFileSync(GENERATED_TS_PATH, generateSource(catalog));
  console.log(`wrote ${catalog.size} event types to ${GENERATED_TS_PATH}`);
}

if (import.meta.main) main();
