import * as Application from "expo-application";
import * as Device from "expo-device";

import type { ConnectDevice } from "@/models/Connect";

// The model becomes the session's label on the web Devices tab.
export const deviceInfo = (): ConnectDevice => ({
  model: Device.modelName ?? "Phone",
  // Some vendors report a build fingerprint as osName, so the platform is named outright.
  os: ["Android", Device.osVersion].filter(Boolean).join(" "),
  app_version: Application.nativeApplicationVersion ?? "dev",
});
