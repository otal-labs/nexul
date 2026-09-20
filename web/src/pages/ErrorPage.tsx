import { ArrowLeft } from "lucide-react";
import { Link, useNavigate, useRouteError } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { NotFoundIllustration } from "@/components/NotFoundIllustration";
import { Button } from "@/components/ui/button";

// Doubles as the "*" not-found route and the errorElement; useRouteError() returning non-null tells them apart.
export const ErrorPage = () => {
  const error = useRouteError();
  const navigate = useNavigate();
  const isCrash = error != null;

  return (
    <div className="blueprint-bg min-h-screen">
      <div className="flex min-h-screen flex-col items-center gap-12 px-6 py-16 sm:flex-col-reverse sm:justify-center sm:gap-16 sm:px-8">
        <div className="flex max-w-3xl flex-col items-center gap-3 text-center">
          <p className="font-mono text-[11px] font-medium tracking-[0.24em] text-primary/90 uppercase">
            Nexul
          </p>
          <h1 className="text-4xl font-semibold tracking-tight text-balance sm:text-6xl">
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
          <div className="flex w-full flex-col gap-1.5 sm:w-fit sm:flex-row">
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
                <ArrowLeft />
                Go back
              </Button>
            )}
          </div>
        </div>
        <NotFoundIllustration />
      </div>
    </div>
  );
};
