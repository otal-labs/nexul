const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

// The barrel evaluates every icon (about 1,900 modules) at startup, since Expo's tree shaking is off.
const lucideByPath = { name: "lucide-react-native", message: "Import each icon by path: lucide-react-native/icons/<name>.", allowTypeImports: true };
// Metro has no path to the SDK, so the app takes the event catalog's types and never its code; tests may read its fixtures.
const sdkTypesOnly = { name: "@otal-labs/nexul-sdk/events", message: "Import the event catalog with `import type`.", allowTypeImports: true };
const noElse = { selector: "IfStatement[alternate]", message: "Early return, no else." };
const arbitraryTextSize = String.raw`/(^|[\s"'\x60:])text-\[\d+(\.\d+)?(px|rem)\]/`;
const typeScaleMessage = "Use a step of the type scale in global.css (text-micro, text-row, text-body…), not an arbitrary text size.";

module.exports = defineConfig([
  expoConfig,
  {
    // eslint-plugin-react's "detect" calls context.getFilename, which ESLint 10 removed.
    settings: { react: { version: require("react/package.json").version } },
  },
  {
    rules: { "no-restricted-imports": ["error", { paths: [lucideByPath, sdkTypesOnly] }] },
  },
  {
    files: ["**/*.test.ts", "**/*.test.tsx"],
    rules: { "no-restricted-imports": ["error", { paths: [lucideByPath] }] },
  },
  // Rule 7: early return, no else. Rule 8: no console.log in production code.
  {
    rules: {
      "no-console": ["error", { allow: ["warn", "error"] }],
      "no-restricted-syntax": ["error", noElse],
    },
  },
  // Sizes come from the type scale in global.css.
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": ["error", noElse,
        { selector: `Literal[value=${arbitraryTextSize}]`, message: typeScaleMessage },
        { selector: `TemplateElement[value.raw=${arbitraryTextSize}]`, message: typeScaleMessage },
      ],
    },
  },
  {
    files: ["**/*.test.ts", "**/*.test.tsx"],
    rules: { "no-console": "off" },
  },
  {
    ignores: ["android/*", "ios/*", ".expo/*", "coverage/*", "src/uniwind-types.d.ts"],
  },
]);
