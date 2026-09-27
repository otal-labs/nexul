import { Button } from "@/components/ui/button";

// Rendered outside the router on first run, so a plain link rather than react-router's Link.
export const SetupDone = () => (
  <Button asChild className="w-full">
    <a href="/login">Sign in</a>
  </Button>
);
