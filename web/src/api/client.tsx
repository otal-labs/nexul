import axios, { type AxiosError } from "axios";

import { useSessionStore } from "@/stores/sessionStore";

export interface ApiErrorBody {
  message: string;
  code: string;
  errors?: Record<string, string[]>;
  // Mirrors httpx.Envelope.Details: the error's structured side, shaped per error (for example pairing.RefusalDetails).
  details?: unknown;
}

const DEFAULT_API_URL = "http://localhost:8080";

export const API_BASE_URL = import.meta.env.VITE_API_URL || DEFAULT_API_URL;

// Strips the trailing slash so API_BASE_URL="/" doesn't produce the protocol-relative "//auth/github".
export const joinAPIURL = (path: string) => `${API_BASE_URL.replace(/\/+$/, "")}${path}`;

// Evaluated at call time so VITE_WS_URL test stubs take effect, not just at module load.
export const resolveWSBase = () =>
  /^https?:\/\//.test(API_BASE_URL)
    ? API_BASE_URL.replace(/^http/, "ws")
    : `${window.location.protocol === "https:" ? "wss" : "ws"}://${window.location.host}`;

export const api = axios.create({ baseURL: API_BASE_URL });

api.interceptors.request.use((config) => {
  const token = useSessionStore.getState().token;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

const redirectToLogin = () => {
  if (window.location.pathname !== "/login") {
    window.location.assign("/login");
  }
};

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiErrorBody>) => {
    if (error.response?.status === 401) {
      useSessionStore.getState().logout();
      redirectToLogin();
    }
    return Promise.reject(error);
  },
);

export const errorMessage = (error: unknown): string => {
  const e = error as AxiosError<ApiErrorBody>;
  const body = e?.response?.data;
  const firstField = body?.errors ? Object.values(body.errors)[0]?.[0] : undefined;
  return firstField || body?.message || e?.message || "Something went wrong";
};
