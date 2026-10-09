const { resolveBabelOptions } = require("jest-expo/src/resolveBabelOptions");

module.exports = {
  preset: "jest-expo",
  setupFiles: ["<rootDir>/jest.setup.ts"],
  // The app imports the SDK's event catalog for its types only; tests also read its topic list and fixtures.
  // A shared module's own imports (Babel's helpers) resolve from this app's node_modules.
  moduleDirectories: ["node_modules", "<rootDir>/node_modules"],
  moduleNameMapper: {
    "^@nexul/sdk/events$": "<rootDir>/../sdk/src/events.generated.ts",
    "^@nexul/client-core/(.*)$": "<rootDir>/../client-core/$1",
  },
  transformIgnorePatterns: [
    "/node_modules/(?!(.pnpm|react-native|@react-native|@react-native-community|expo|@expo|@expo-google-fonts|react-navigation|@react-navigation|@sentry/react-native|native-base|standard-navigation|uniwind|@rn-primitives|lucide-react-native|react-native-svg))",
  ],
  // lucide-react-native ships .mjs; jest-expo's own transform only targets .js/.ts/.tsx, so this reuses the same babel options for that extension.
  transform: {
    "\\.mjs$": ["babel-jest", resolveBabelOptions()],
  },
};
