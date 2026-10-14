import Constants from "expo-constants";
import * as Notifications from "expo-notifications";

import { requires } from "@nexul/client-core/requires";

import { requestJSON } from "@/api/client";

const CHANNEL_ID = "default";

// Set by the owner's one-time Expo/Firebase setup (ticket 13); empty until then.
const pushProjectId = (): string | undefined => {
  const id = Constants.expoConfig?.extra?.eas?.projectId;
  return typeof id === "string" && id.length > 0 ? id : undefined;
};

const ensureChannel = () =>
  Notifications.setNotificationChannelAsync(CHANNEL_ID, { name: "Nexul", importance: Notifications.AndroidImportance.DEFAULT });

const ensurePermission = async (): Promise<boolean> => {
  const current = await Notifications.getPermissionsAsync();
  if (current.granted) return true;
  const requested = await Notifications.requestPermissionsAsync();
  return requested.granted;
};

const putPushToken = requires("none", (host: string, token: string, pushToken: string) =>
  requestJSON(`${host}/api/auth/sessions/current/push-token`, { method: "PUT", body: { push_token: pushToken }, token }));

// A missing project id or a denied permission skips registration, logs once, never nags.
export const registerPushToken = async (host: string, token: string): Promise<void> => {
  const projectId = pushProjectId();
  if (!projectId) {
    console.log("push: no project id configured, skipping registration");
    return;
  }
  try {
    await ensureChannel();
    if (!(await ensurePermission())) {
      console.log("push: permission not granted, skipping registration");
      return;
    }
    const expoPushToken = await Notifications.getExpoPushTokenAsync({ projectId });
    await putPushToken(host, token, expoPushToken.data);
  } catch (error) {
    console.warn("push: failed to register token", error);
  }
};

// Best-effort: called while the session is still valid, before it is deleted.
export const clearPushToken = async (host: string, token: string): Promise<void> => {
  try {
    await putPushToken(host, token, "");
  } catch (error) {
    console.warn("push: failed to clear token", error);
  }
};
