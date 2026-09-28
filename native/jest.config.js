const { resolveBabelOptions } = require("jest-expo/src/resolveBabelOptions");

module.exports = {
  preset: "jest-expo",
  transformIgnorePatterns: [
    "/node_modules/(?!(.pnpm|react-native|@react-native|@react-native-community|expo|@expo|@expo-google-fonts|react-navigation|@react-navigation|@sentry/react-native|native-base|standard-navigation|uniwind|@rn-primitives|lucide-react-native|react-native-svg))",
  ],
  // lucide-react-native ships .mjs; jest-expo's own transform only targets .js/.ts/.tsx, so this reuses the same babel options for that extension.
  transform: {
    "\\.mjs$": ["babel-jest", resolveBabelOptions()],
  },
};
