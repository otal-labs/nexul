// Config knobs declared in code: the platform renders these as a real
// form (status dropdown, channel picker, plain text), never a raw JSON blob.
export interface ConfigField {
  // "status" and "channel" are semantic string types the settings UI renders
  // as workspace pickers; they ride the wire as JSON Schema
  // `type:"string"` + `format`.
  type: "string" | "number" | "boolean" | "status" | "channel";
  label?: string;
  default?: string | number | boolean;
  enum?: string[];
  required?: boolean;
}

export type ConfigSchema = Record<string, ConfigField>;

type FieldValue<F extends ConfigField> = F["enum"] extends string[]
  ? F["enum"][number]
  : F["type"] extends "string" | "status" | "channel"
    ? string
    : F["type"] extends "number"
      ? number
      : boolean;

export type ConfigValues<S extends ConfigSchema> = { [K in keyof S]: FieldValue<S[K]> };

// toJsonSchema renders the code-declared config as the JSON Schema the
// announce frame's config_schema field carries — the wire shape a
// generic settings-page renderer reads, independent of this SDK's own
// TS-side ConfigField shape.
export function toJsonSchema(schema: ConfigSchema): unknown {
  const properties: Record<string, unknown> = {};
  const required: string[] = [];
  for (const [name, field] of Object.entries(schema)) {
    const semantic = field.type === "status" || field.type === "channel";
    properties[name] = {
      type: semantic ? "string" : field.type,
      ...(semantic ? { format: field.type } : {}),
      ...(field.label !== undefined ? { title: field.label } : {}),
      ...(field.default !== undefined ? { default: field.default } : {}),
      ...(field.enum !== undefined ? { enum: field.enum } : {}),
    };
    if (field.required) required.push(name);
  }
  return { type: "object", properties, ...(required.length > 0 ? { required } : {}) };
}

export function defaultConfigValues<S extends ConfigSchema>(schema: S): ConfigValues<S> {
  const out: Record<string, unknown> = {};
  for (const [name, field] of Object.entries(schema)) {
    out[name] =
      field.default ?? (field.type === "number" ? 0 : field.type === "boolean" ? false : "");
  }
  return out as ConfigValues<S>;
}
