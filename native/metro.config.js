const path = require("path");
const { getDefaultConfig } = require("expo/metro-config");
const { withUniwindConfig } = require("uniwind/metro");

const config = getDefaultConfig(__dirname);

// The modules shared with the web app live beside native/; tsconfig's paths name them, and Metro watches the folder.
config.watchFolders = [...config.watchFolders, path.resolve(__dirname, "../client-core")];

module.exports = withUniwindConfig(config, {
  cssEntryFile: "./src/global.css",
  dtsFile: "./src/uniwind-types.d.ts",
});
