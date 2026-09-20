import { common, createLowlight } from "lowlight";

// Shared lowlight instance (createLowlight is a factory, not a singleton) using the "common" grammar bundle.
export const lowlight = createLowlight(common);

export interface CodeLanguageOption {
  value: string;
  label: string;
}

const LABEL_OVERRIDES: Record<string, string> = {
  csharp: "C#",
  cpp: "C++",
  objectivec: "Objective-C",
  javascript: "JavaScript",
  typescript: "TypeScript",
  json: "JSON",
  yaml: "YAML",
  sql: "SQL",
  html: "HTML",
  css: "CSS",
  scss: "SCSS",
  xml: "XML",
  php: "PHP",
  vbnet: "VB.NET",
  wasm: "WebAssembly",
  ini: "INI",
  graphql: "GraphQL",
};

function languageLabel(value: string): string {
  return LABEL_OVERRIDES[value] ?? value.charAt(0).toUpperCase() + value.slice(1);
}

// Sorted alphabetically by label so the dropdown is scannable instead of grammar-registration order.
export const codeLanguageOptions: CodeLanguageOption[] = [
  { value: "plaintext", label: "Plain text" },
  ...lowlight
    .listLanguages()
    .filter((value) => value !== "plaintext")
    .map((value) => ({ value, label: languageLabel(value) }))
    .sort((a, b) => a.label.localeCompare(b.label)),
];
