import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { ContextAwareConfirmation } from "react-confirm";
import { createMemoryRouter, RouterProvider } from "react-router";

import { PlaysPage } from "@/pages/PlaysPage";

// The page at its own routes, in a data router since the auto play composer guards leaving it; the router is returned to read where a click went.
export const renderPlaysPage = (path = "/acme/plays") => {
  const router = createMemoryRouter(
    [
      { path: "/acme/plays", element: <PlaysPage /> },
      { path: "/acme/plays/:playId", element: <PlaysPage /> },
    ],
    { initialEntries: [path] },
  );
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
  return router;
};
