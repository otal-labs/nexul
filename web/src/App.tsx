import { QueryClientProvider } from "@tanstack/react-query";
import { LucideProvider } from "lucide-react";
import { ContextAwareConfirmation } from "react-confirm";

import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { queryClient } from "@/lib/queryClient";
import { AppRouter } from "@/Router";

export const App = () => (
  <LucideProvider strokeWidth={1.75}>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <ContextAwareConfirmation.ConfirmationRoot />
        <AppRouter />
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  </LucideProvider>
);
