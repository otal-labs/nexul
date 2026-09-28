import { networkOnlineListener } from "@/lib/onlineStatus";

type Listener = (state: { isConnected: boolean | null }) => void;
const mockListeners: Listener[] = [];

jest.mock("expo-network", () => ({
  addNetworkStateListener: (listener: Listener) => {
    mockListeners.push(listener);
    return { remove: () => mockListeners.splice(mockListeners.indexOf(listener), 1) };
  },
  getNetworkStateAsync: () => Promise.reject(new Error("unavailable")),
}));

describe("networkOnlineListener", () => {
  test("reports the network state, treats unknown as offline, and unsubscribes on cleanup", () => {
    const seen: boolean[] = [];
    const cleanup = networkOnlineListener((online) => seen.push(online));
    expect(mockListeners).toHaveLength(1);

    mockListeners[0]?.({ isConnected: false });
    mockListeners[0]?.({ isConnected: null });
    mockListeners[0]?.({ isConnected: true });
    expect(seen).toEqual([false, false, true]);

    cleanup();
    expect(mockListeners).toHaveLength(0);
  });
});
