import * as SecureStore from "expo-secure-store";
import { create } from "zustand";

import { queryClient } from "@/lib/queryClient";

const HOST_KEY = "session_host";
const TOKEN_KEY = "session_token";

export type SessionStore = {
  host: string | null;
  signedIn: boolean;
  signIn: (host: string, token: string) => void;
  signOut: () => void;
};

// The token stays in the secure store and is read on demand; the store only knows that one exists.
export const readSessionToken = (): string | null => SecureStore.getItem(TOKEN_KEY);

export const useSessionStore = create<SessionStore>((set) => ({
  host: SecureStore.getItem(HOST_KEY),
  signedIn: SecureStore.getItem(TOKEN_KEY) !== null,
  signIn: (host, token) => {
    SecureStore.setItem(HOST_KEY, host);
    SecureStore.setItem(TOKEN_KEY, token);
    set({ host, signedIn: true });
  },
  signOut: () => {
    set({ host: null, signedIn: false });
    queryClient.clear();
    void SecureStore.deleteItemAsync(HOST_KEY);
    void SecureStore.deleteItemAsync(TOKEN_KEY);
  },
}));
