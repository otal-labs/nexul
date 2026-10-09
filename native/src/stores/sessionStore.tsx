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

// Read from the secure store once at launch: a read decrypts on the JS thread, and every request needs the token.
let sessionToken = SecureStore.getItem(TOKEN_KEY);

export const readSessionToken = (): string | null => sessionToken;

export const useSessionStore = create<SessionStore>((set) => ({
  host: SecureStore.getItem(HOST_KEY),
  signedIn: sessionToken !== null,
  signIn: (host, token) => {
    SecureStore.setItem(HOST_KEY, host);
    SecureStore.setItem(TOKEN_KEY, token);
    sessionToken = token;
    set({ host, signedIn: true });
  },
  signOut: () => {
    sessionToken = null;
    set({ host: null, signedIn: false });
    queryClient.clear();
    void SecureStore.deleteItemAsync(HOST_KEY);
    void SecureStore.deleteItemAsync(TOKEN_KEY);
  },
}));
