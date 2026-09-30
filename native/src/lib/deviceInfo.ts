import * as Application from "expo-application";
import * as Device from "expo-device";
import { Platform } from "react-native";

import type { ConnectDevice } from "@/models/Connect";

// Some vendors report a build fingerprint as osName, so the platform is named outright.
const platformName: Record<string, string> = { ios: "iOS", android: "Android" };

// The model becomes the session's label on the web Devices tab.
export const deviceInfo = (): ConnectDevice => ({
  model: Device.modelName ?? "Phone",
  os: [platformName[Platform.OS] ?? "Android", Device.osVersion].filter(Boolean).join(" "),
  app_version: Application.nativeApplicationVersion ?? "dev",
});
