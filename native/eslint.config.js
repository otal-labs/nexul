const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    // eslint-plugin-react's "detect" calls context.getFilename, which ESLint 10 removed.
    settings: { react: { version: require("react/package.json").version } },
  },
  {
    ignores: ["android/*", "ios/*", ".expo/*", "coverage/*", "src/uniwind-types.d.ts"],
  },
]);
