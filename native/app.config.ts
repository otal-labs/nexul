import { existsSync } from "node:fs";
import type { ExpoConfig } from "expo/config";

// The update server's certificate, downloaded from its dashboard once the app exists there; until then updates are unsigned.
const certificate = "./certs/certificate.pem";

// Firebase's FCM credentials file, path supplied by the build environment; until the owner runs the push setup, builds go without it.
const googleServicesFile = process.env.NEXUL_GOOGLE_SERVICES_JSON;

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

const version = "0.1.4";
// Android only installs an update with a higher build number, so derive it from the version and every release climbs.
const [major = 0, minor = 0, patch = 0] = version.split(".").map(Number);
const buildNumber = major * 10000 + minor * 100 + patch;

const config: ExpoConfig = {
  name: "Nexul",
  slug: "nexul",
  version,
  scheme: "nexul",
  orientation: "portrait",
  userInterfaceStyle: "automatic",
  ios: {
    bundleIdentifier: "io.nexul.app",
    buildNumber: String(buildNumber),
    infoPlist: { ITSAppUsesNonExemptEncryption: false },
  },
  android: {
    package: "io.nexul.app",
    versionCode: buildNumber,
    ...(googleServicesFile && existsSync(googleServicesFile) && { googleServicesFile }),
  },
  runtimeVersion: { policy: "appVersion" },
  updates: {
    url: process.env.NEXUL_UPDATES_URL ?? "",
    requestHeaders: {
      "expo-channel-name": "production",
      "expo-app-id": process.env.NEXUL_UPDATES_APP_ID ?? "",
    },
    ...(existsSync(certificate) && {
      codeSigningCertificate: certificate,
      codeSigningMetadata: { keyid: "main", alg: "rsa-v1_5-sha256" },
    }),
  },
  extra: { eas: { projectId: process.env.NEXUL_EAS_PROJECT_ID ?? "" } },
  plugins: [
    "expo-router",
    "./plugins/withReleaseSigning",
    "expo-secure-store",
    "expo-notifications",
    [
      "expo-camera",
      {
        cameraPermission: "Nexul scans the connect code shown on your Devices page.",
        microphonePermission: false,
        recordAudioAndroid: false,
      },
    ],
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
