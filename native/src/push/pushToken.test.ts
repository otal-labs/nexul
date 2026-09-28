import * as Notifications from "expo-notifications";

import { clearPushToken, registerPushToken } from "@/push/pushToken";

jest.mock("expo-constants", () => {
  const mockConstants = { expoConfig: { extra: { eas: { projectId: "proj_1" } } } };
  return { __esModule: true, mockConstants, default: mockConstants };
});

jest.mock("expo-notifications", () => ({
  AndroidImportance: { DEFAULT: 3 },
  setNotificationChannelAsync: jest.fn(async () => null),
  getPermissionsAsync: jest.fn(async () => ({ granted: true })),
  requestPermissionsAsync: jest.fn(async () => ({ granted: true })),
  getExpoPushTokenAsync: jest.fn(async () => ({ data: "ExponentPushToken[abc]" })),
}));

const mockConstants = (
  jest.requireMock("expo-constants") as {
    mockConstants: { expoConfig: { extra: { eas: { projectId: string } } } };
  }
).mockConstants;

const mockFetch = jest.fn();
globalThis.fetch = mockFetch;

const respond = (status: number, body: unknown) =>
  mockFetch.mockResolvedValue({ ok: status < 300, status, text: async () => JSON.stringify(body) });

describe("pushToken", () => {
  beforeEach(() => {
    mockFetch.mockReset();
    mockConstants.expoConfig.extra.eas.projectId = "proj_1";
    jest.mocked(Notifications.getPermissionsAsync).mockResolvedValue({ granted: true } as never);
    jest.mocked(Notifications.requestPermissionsAsync).mockResolvedValue({ granted: true } as never);
  });

  test("registerPushToken PUTs the Expo token once permission is granted", async () => {
    respond(204, null);

    await registerPushToken("https://nexul.example.com", "ses_abc");

    expect(mockFetch).toHaveBeenCalledWith("https://nexul.example.com/api/auth/sessions/current/push-token", {
      method: "PUT",
      headers: { Accept: "application/json", "Content-Type": "application/json", Authorization: "Bearer ses_abc" },
      body: JSON.stringify({ push_token: "ExponentPushToken[abc]" }),
    });
  });

  test("registerPushToken skips without a project id, never reaching the network", async () => {
    mockConstants.expoConfig.extra.eas.projectId = "";

    await registerPushToken("https://nexul.example.com", "ses_abc");

    expect(mockFetch).not.toHaveBeenCalled();
  });

  test("registerPushToken skips when permission is denied, never reaching the network", async () => {
    jest.mocked(Notifications.getPermissionsAsync).mockResolvedValue({ granted: false } as never);
    jest.mocked(Notifications.requestPermissionsAsync).mockResolvedValue({ granted: false } as never);

    await registerPushToken("https://nexul.example.com", "ses_abc");

    expect(mockFetch).not.toHaveBeenCalled();
  });

  test("registerPushToken never throws when the PUT fails", async () => {
    mockFetch.mockRejectedValue(new Error("network down"));

    await expect(registerPushToken("https://nexul.example.com", "ses_abc")).resolves.toBeUndefined();
  });

  test("clearPushToken PUTs an empty token", async () => {
    respond(204, null);

    await clearPushToken("https://nexul.example.com", "ses_abc");

    expect(mockFetch).toHaveBeenCalledWith("https://nexul.example.com/api/auth/sessions/current/push-token", {
      method: "PUT",
      headers: { Accept: "application/json", "Content-Type": "application/json", Authorization: "Bearer ses_abc" },
      body: JSON.stringify({ push_token: "" }),
    });
  });

  test("clearPushToken never throws when the PUT fails", async () => {
    mockFetch.mockRejectedValue(new Error("network down"));

    await expect(clearPushToken("https://nexul.example.com", "ses_abc")).resolves.toBeUndefined();
  });
});
