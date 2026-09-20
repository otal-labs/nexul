import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";

import { Toaster } from "@/components/ui/sonner";
import { AppRouter } from "@/Router";

// staleTime 30s: useLiveEvents already pushes invalidations on change, so refetching at 0 was redundant.
const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } });

export const App = () => (
  <QueryClientProvider client={queryClient}>
    <ContextAwareConfirmation.ConfirmationRoot />
    <AppRouter />
    <Toaster />
  </QueryClientProvider>
);
