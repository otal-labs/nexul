import { QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";

import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { queryClient } from "@/lib/queryClient";
import { AppRouter } from "@/Router";

export const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <ContextAwareConfirmation.ConfirmationRoot />
      <AppRouter />
      <Toaster />
    </TooltipProvider>
  </QueryClientProvider>
);
