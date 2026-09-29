import { render, screen } from "@testing-library/react-native";

import { ApiError } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";

const notFoundError = new ApiError(404, { message: "get doc x: not found", code: "not_found" }, "GET failed: 404");

describe("ErrorDisplay", () => {
  test("shows the not-found message instead of the raw one on a 404", async () => {
    await render(<ErrorDisplay error={notFoundError} notFound="This doc doesn't exist or was deleted." />);
    expect(screen.getByText("This doc doesn't exist or was deleted.")).toBeTruthy();
    expect(screen.queryByText(/not found/)).toBeNull();
  });

  test("keeps the server message for a 404 when the screen gave no not-found text", async () => {
    await render(<ErrorDisplay error={notFoundError} />);
    expect(screen.getByText("get doc x: not found")).toBeTruthy();
  });

  test("keeps the server message for any other status", async () => {
    const error = new ApiError(500, { message: "database is locked", code: "internal" }, "GET failed: 500");
    await render(<ErrorDisplay error={error} notFound="This doc doesn't exist or was deleted." />);
    expect(screen.getByText("database is locked")).toBeTruthy();
  });
});
