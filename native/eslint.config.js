const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

// The barrel evaluates every icon (about 1,900 modules) at startup, since Expo's tree shaking is off.
const lucideByPath = { name: "lucide-react-native", message: "Import each icon by path: lucide-react-native/icons/<name>.", allowTypeImports: true };
// Metro has no path to the SDK, so the app takes the event catalog's types and never its code; tests may read its fixtures.
const sdkTypesOnly = { name: "@nexul/sdk/events", message: "Import the event catalog with `import type`.", allowTypeImports: true };

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
  {
    ignores: ["android/*", "ios/*", ".expo/*", "coverage/*", "src/uniwind-types.d.ts"],
  },
]);
