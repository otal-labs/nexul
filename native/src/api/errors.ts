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

// A 403 is the server refusing an item the viewer can't see; to the viewer it is as missing as a 404.
export const isNotFound = (error: unknown): boolean =>
  error instanceof ApiError && (error.status === 404 || error.status === 403);

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
