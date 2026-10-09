const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    // eslint-plugin-react's "detect" calls context.getFilename, which ESLint 10 removed.
    settings: { react: { version: require("react/package.json").version } },
  },
  {
    rules: {
      // The barrel evaluates every icon (about 1,900 modules) at startup, since Expo's tree shaking is off.
      "no-restricted-imports": [
        "error",
        { paths: [{ name: "lucide-react-native", message: 'Import each icon by path: lucide-react-native/icons/<name>.', allowTypeImports: true }] },
      ],
    },
  },
  {
    ignores: ["android/*", "ios/*", ".expo/*", "coverage/*", "src/uniwind-types.d.ts"],
  },
]);
