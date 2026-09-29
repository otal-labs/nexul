import { ApiError } from "@/api/errors";
import { exchangeConnectCode, fetchAbout } from "@/api/connect";
import { connectPhone } from "@/hooks/ConnectHooks";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    getItem: (key: string) => secrets.get(key) ?? null,
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

jest.mock("@/api/connect", () => ({ fetchAbout: jest.fn(), exchangeConnectCode: jest.fn() }));

jest.mock("@/lib/deviceInfo", () => ({
  deviceInfo: () => ({ model: "Pixel 9", os: "Android 16", app_version: "0.1.1" }),
}));

const about = jest.mocked(fetchAbout);
const exchange = jest.mocked(exchangeConnectCode);
const host = "https://nexul.example.com";
const input = { host, code: "ABCD-EFGH-JKMN" };

describe("connectPhone", () => {
  beforeEach(() => {
    about.mockReset();
    exchange.mockReset();
    useSessionStore.getState().signOut();
  });

  test("an address that is not a Nexul server fails before any code is sent", async () => {
    about.mockRejectedValue(new Error(`${host} is not a Nexul server.`));

    await expect(connectPhone(input)).rejects.toThrow("is not a Nexul server");
    expect(exchange).not.toHaveBeenCalled();
  });

  test("an older server is refused without spending the code", async () => {
    about.mockResolvedValue({ product: "nexul", version: "v0.1.0" });

    await expect(connectPhone(input)).resolves.toEqual({ kind: "refused", host, version: "v0.1.0" });
    expect(exchange).not.toHaveBeenCalled();
    expect(useSessionStore.getState().signedIn).toBe(false);
  });

  test("a wrong or expired code gives one clear message", async () => {
    about.mockResolvedValue({ product: "nexul", version: "dev" });
    exchange.mockRejectedValue(new ApiError(400, { message: "invalid", code: "invalid_code" }, "POST failed: 400"));

    await expect(connectPhone(input)).rejects.toThrow("That code is invalid or expired. Get a new one from Devices.");
    expect(useSessionStore.getState().signedIn).toBe(false);
  });

  test("the rate limit says to wait", async () => {
    about.mockResolvedValue({ product: "nexul", version: "dev" });
    exchange.mockRejectedValue(new ApiError(429, { message: "slow down", code: "RATE_LIMITED" }, "POST failed: 429"));

    await expect(connectPhone(input)).rejects.toThrow("Too many wrong codes. Try again in a few minutes.");
  });

  test("a supported server trades the code for a session kept in the secure store", async () => {
    about.mockResolvedValue({ product: "nexul", version: "v9.0.0" });
    exchange.mockResolvedValue({ token: "ses_new", server_version: "v9.0.0" });

    await expect(connectPhone(input)).resolves.toEqual({ kind: "connected" });
    expect(exchange).toHaveBeenCalledWith(host, "ABCD-EFGH-JKMN", {
      model: "Pixel 9",
      os: "Android 16",
      app_version: "0.1.1",
    });
    expect(useSessionStore.getState()).toMatchObject({ host, signedIn: true });
    expect(readSessionToken()).toBe("ses_new");
  });
});
