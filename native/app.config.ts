import type { ExpoConfig } from "expo/config";

const inter = "node_modules/@expo-google-fonts/inter";
const mono = "node_modules/@expo-google-fonts/jetbrains-mono";

// Android needs the family declared once with its weights; iOS reads the family from the files.
const fonts = [
  {
    fontFamily: "Inter",
    fontDefinitions: [
      { path: `./${inter}/400Regular/Inter_400Regular.ttf`, weight: 400 },
      { path: `./${inter}/500Medium/Inter_500Medium.ttf`, weight: 500 },
      { path: `./${inter}/600SemiBold/Inter_600SemiBold.ttf`, weight: 600 },
    ],
  },
  {
    fontFamily: "JetBrains Mono",
    fontDefinitions: [
      { path: `./${mono}/400Regular/JetBrainsMono_400Regular.ttf`, weight: 400 },
      { path: `./${mono}/500Medium/JetBrainsMono_500Medium.ttf`, weight: 500 },
    ],
  },
];

const config: ExpoConfig = {
  name: "Nexul",
  slug: "nexul",
  version: "0.1.0",
  scheme: "nexul",
  orientation: "portrait",
  userInterfaceStyle: "automatic",
  android: {
    package: "io.nexul.app",
  },
  runtimeVersion: { policy: "appVersion" },
  updates: { url: process.env.NEXUL_UPDATES_URL ?? "" },
  extra: { eas: { projectId: process.env.NEXUL_EAS_PROJECT_ID ?? "" } },
  plugins: [
    "expo-router",
    [
      "expo-font",
      {
        android: { fonts },
        ios: { fonts: fonts.flatMap((f) => f.fontDefinitions.map((d) => d.path)) },
      },
    ],
  ],
  experiments: { typedRoutes: true, reactCompiler: true },
};

export default config;
