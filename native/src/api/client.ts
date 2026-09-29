import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

export interface ApiErrorBody {
  message: string;
  code: string;
  errors?: Record<string, string[]>;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: unknown,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export const isNotFound = (error: unknown): boolean => error instanceof ApiError && error.status === 404;

const isErrorBody = (body: unknown): body is ApiErrorBody =>
  typeof body === "object" && body !== null && typeof (body as ApiErrorBody).message === "string";

export const errorMessage = (error: unknown): string => {
  if (error instanceof ApiError && isErrorBody(error.body)) {
    const firstField = error.body.errors ? Object.values(error.body.errors)[0]?.[0] : undefined;
    return firstField || error.body.message;
  }
  if (error instanceof Error && error.message) return error.message;
  return "Something went wrong";
};

const parseBody = async (response: Response): Promise<unknown> => {
  const text = await response.text();
  if (text === "") return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
};

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export interface RequestOptions {
  method?: Method;
  body?: unknown;
  token?: string;
}

// The one fetch every call goes through: JSON in and out, a bearer when there is one, ApiError on a non-2xx.
export const requestJSON = async <T>(url: string, options: RequestOptions = {}): Promise<T> => {
  const method = options.method ?? "GET";
  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (options.token) headers.Authorization = `Bearer ${options.token}`;
  const response = await fetch(url, {
    method,
    headers,
    body: options.body === undefined ? null : JSON.stringify(options.body),
  });
  const body = await parseBody(response);
  if (!response.ok) throw new ApiError(response.status, body, `${method} ${url} failed: ${response.status}`);
  return body as T;
};

const call = async <T>(method: Method, path: string, body?: unknown): Promise<T> => {
  const { host } = useSessionStore.getState();
  const token = readSessionToken();
  if (!host || !token) throw new ApiError(401, null, `${method} ${path} failed: signed out`);
  try {
    return await requestJSON<T>(`${host}${path}`, { method, body, token });
  } catch (error) {
    // A 401 from any call means the session is gone: back to first-run, cache cleared.
    if (error instanceof ApiError && error.status === 401) useSessionStore.getState().signOut();
    throw error;
  }
};

export const api = {
  get: <T>(path: string) => call<T>("GET", path),
  post: <T>(path: string, body?: unknown) => call<T>("POST", path, body),
  put: <T>(path: string, body?: unknown) => call<T>("PUT", path, body),
  patch: <T>(path: string, body?: unknown) => call<T>("PATCH", path, body),
  delete: <T>(path: string) => call<T>("DELETE", path),
};
