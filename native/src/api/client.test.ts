import { api, ApiError } from "@/api/client";
import { useSessionStore } from "@/stores/sessionStore";

jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    getItem: (key: string) => secrets.get(key) ?? null,
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

const mockFetch = jest.fn();
globalThis.fetch = mockFetch;

const respond = (status: number, body: unknown) =>
  mockFetch.mockResolvedValue({ ok: status < 300, status, text: async () => JSON.stringify(body) });

describe("api", () => {
  beforeEach(() => {
    mockFetch.mockReset();
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
  });

  test("a 401 from any call signs the phone out", async () => {
    respond(401, { message: "unauthorized", code: "UNAUTHORIZED" });

    await expect(api.get("/api/auth/me")).rejects.toBeInstanceOf(ApiError);
    expect(useSessionStore.getState()).toMatchObject({ host: null, signedIn: false });
  });

  test("any other failure keeps the session", async () => {
    respond(500, { message: "boom", code: "INTERNAL" });

    await expect(api.get("/api/auth/me")).rejects.toMatchObject({ status: 500 });
    expect(useSessionStore.getState().signedIn).toBe(true);
  });

  test("a signed-out call fails without reaching the network", async () => {
    useSessionStore.getState().signOut();

    await expect(api.get("/api/auth/me")).rejects.toMatchObject({ status: 401 });
    expect(mockFetch).not.toHaveBeenCalled();
  });

  test("calls go to the stored host with the bearer token", async () => {
    respond(200, { user: { id: "u1" } });

    await expect(api.post("/api/things", { a: 1 })).resolves.toEqual({ user: { id: "u1" } });
    expect(mockFetch).toHaveBeenCalledWith("https://nexul.example.com/api/things", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json", Authorization: "Bearer ses_abc" },
      body: '{"a":1}',
    });
  });
});
