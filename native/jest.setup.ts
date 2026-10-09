import "react-native-gesture-handler/jestSetup";
import { setUpTests } from "react-native-reanimated";

// Native modules every screen test reaches through the app's stores and query client; tests may still override them.
jest.mock("expo-sqlite/kv-store", () => {
  const store = new Map<string, string>();
  return {
    __esModule: true,
    default: {
      getItemSync: (key: string) => store.get(key) ?? null,
      setItemSync: (key: string, value: string) => store.set(key, value),
      removeItemSync: (key: string) => store.delete(key),
    },
  };
});

jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));

// Worklets need the native runtime; their mock runs every animation to its end state synchronously.
jest.mock("react-native-worklets", () => jest.requireActual("react-native-worklets/lib/module/mock"));
setUpTests();

// Screens read the status bar inset; the library's own mock gives every inset as zero.
jest.mock("react-native-safe-area-context", () => jest.requireActual("react-native-safe-area-context/jest/mock").default);
