import js from "@eslint/js";
import tanstackQuery from "@tanstack/eslint-plugin-query";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

const classRules = [
  {
    pattern: String.raw`/(^|[\s"'\x60:])text-\[\d+(\.\d+)?(px|rem)\]/`,
    message: "Use a step of the type scale (text-micro, text-xs, text-row, text-sm…), not an arbitrary text size.",
  },
  {
    pattern: String.raw`/(^|[\s"'\x60:])font-\[\d+\]/`,
    message: "Use a named font weight (font-book, font-medium…), not an arbitrary one.",
  },
  {
    pattern: String.raw`/(^|[\s"'\x60])focus-visible:(ring|outline)/`,
    message: "The global :focus-visible outline overrides this; drop it, or use quiet-focus or inset-focus.",
  },
];

export default tseslint.config(
  { ignores: ["dist", "coverage"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...tanstackQuery.configs["flat/recommended"],
  {
    files: ["**/*.{ts,tsx}"],
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  },
  // Sizes come from the type scale in index.css, and the global :focus-visible outline is the one indicator.
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": ["error", ...classRules.flatMap(({ pattern, message }) => [
        { selector: `Literal[value=${pattern}]`, message },
        { selector: `TemplateElement[value.raw=${pattern}]`, message },
      ])],
    },
  },
  // shadcn primitives are CLI-generated and export their variants beside the component.
  {
    files: ["src/components/ui/**"],
    rules: { "react-refresh/only-export-components": "off" },
  },
);
