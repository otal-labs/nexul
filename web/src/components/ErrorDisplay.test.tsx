import { QueryClient, QueryClientProvider, QueryObserver } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AxiosError, AxiosHeaders } from "axios";
import { describe, expect, it, vi } from "vitest";

import { ErrorDisplay } from "@/components/ErrorDisplay";

const withStatus = (status: number) =>
  new AxiosError("failed", "ERR_BAD_RESPONSE", undefined, undefined, {
    status,
    statusText: "",
    data: { message: "failed" },
    headers: {},
    config: { headers: new AxiosHeaders() },
  });

const renderPanel = (error: unknown, client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) =>
  render(
    <QueryClientProvider client={client}>
      <ErrorDisplay error={error} />
    </QueryClientProvider>,
  );

describe("ErrorDisplay", () => {
  it.each([
    ["no answer from the server", new AxiosError("Network Error", "ERR_NETWORK"), true],
    ["a server error", withStatus(503), true],
    ["a refusal", withStatus(403), false],
    ["a missing record", withStatus(404), false],
    ["an error thrown in the app", new Error("boom"), false],
  ])("offers Try again only when a second attempt could work: %s", (_name, error, offered) => {
    renderPanel(error);
    expect(!!screen.queryByRole("button", { name: "Try again" })).toBe(offered);
  });

  it("refetches the failed queries on screen when Try again is pressed", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const fetchTickets = vi.fn().mockRejectedValueOnce(new AxiosError("Network Error", "ERR_NETWORK")).mockResolvedValue([]);
    const watch = new QueryObserver(client, { queryKey: ["tickets"], queryFn: fetchTickets });
    const stop = watch.subscribe(() => {});
    await waitFor(() => expect(watch.getCurrentResult().status).toBe("error"));

    renderPanel(watch.getCurrentResult().error, client);
    await userEvent.setup().click(screen.getByRole("button", { name: "Try again" }));

    await waitFor(() => expect(watch.getCurrentResult().status).toBe("success"));
    expect(fetchTickets).toHaveBeenCalledTimes(2);
    stop();
  });
});
