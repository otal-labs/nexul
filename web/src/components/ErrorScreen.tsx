import { Link, useNavigate } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { microheaderClass } from "@/components/Microheader";
import { NotFoundIllustration } from "@/components/NotFoundIllustration";
import { displayTitleClass } from "@/components/PageHeader";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ErrorScreenProps {
  // A route's crash; without one the screen is the app's not-found page.
  error?: unknown;
}

export const ErrorScreen = ({ error }: ErrorScreenProps) => {
  const navigate = useNavigate();
  const isCrash = error != null;

  return (
    <ShowcaseSurface quiet>
      <div className="flex min-h-full flex-col items-center gap-12 px-6 py-16 sm:flex-col-reverse sm:justify-center sm:gap-16 sm:px-8">
        <div className="flex max-w-3xl flex-col items-center gap-3 text-center">
          <p className={microheaderClass}>Nexul</p>
          <h1 className={cn(displayTitleClass, "text-[clamp(2.75rem,6vw,4rem)]")}>
            {isCrash && "Something went wrong"}
            {!isCrash && "Page not found"}
          </h1>
          <p className="max-w-sm font-mono text-sm text-muted-foreground">
            {isCrash && (
              <>
                $ app --resume
                <br />
                <span className="text-destructive">crash</span> — a new version may have shipped; reload to pick it
                up.
              </>
            )}
            {!isCrash && (
              <>
                $ curl /route/that/does/not/exist
                <br />
                <span className="text-destructive">404</span> — nothing listens here.
              </>
            )}
          </p>
          {isCrash && (
            <div className="w-full max-w-sm text-left">
              <ErrorDisplay error={error} />
            </div>
          )}
          <div className="flex w-full flex-col gap-2 sm:w-fit sm:flex-row">
            {isCrash && (
              <Button variant="outline" onClick={() => window.location.reload()}>
                Reload
              </Button>
            )}
            {!isCrash && (
              <Button asChild>
                <Link to="/">Back home</Link>
              </Button>
            )}
            {!isCrash && (
              <Button variant="outline" onClick={() => navigate(-1)}>
                Go back
              </Button>
            )}
          </div>
        </div>
        <NotFoundIllustration />
      </div>
    </ShowcaseSurface>
  );
};
