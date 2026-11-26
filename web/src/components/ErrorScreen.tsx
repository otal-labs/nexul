import { CircleAlert, Unplug } from "lucide-react";
import { Link, useLocation, useNavigate } from "react-router";

import { errorMessage } from "@/api/client";
import { OrbitMark } from "@/components/EmptyState";
import { displayTitleClass } from "@/components/PageHeader";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ErrorScreenProps {
  // A route's crash; without one the screen is the app's not-found page.
  error?: unknown;
}

// The empty state's orbit and its entrance, at the showcase headline's size; the console lines say what happened.
export const ErrorScreen = ({ error }: ErrorScreenProps) => {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const isCrash = error != null;

  return (
    <ShowcaseSurface quiet>
      <div className="flex min-h-full items-center justify-center px-6 pt-16 pb-28 sm:px-8">
        <div data-size="default" className="empty-state flex max-w-md flex-col items-center text-center">
          <OrbitMark icon={isCrash ? CircleAlert : Unplug} compact={false} />
          <h1 className={cn(displayTitleClass, "mt-6 text-headline")}>
            {isCrash && "Something went wrong"}
            {!isCrash && "Page not found"}
          </h1>
          <p role={isCrash ? "alert" : undefined} className="mt-3 max-w-sm font-mono text-xs leading-5 break-words text-muted-foreground">
            {isCrash && (
              <>
                $ app --resume
                <br />
                <span className="text-destructive">crash</span> {errorMessage(error)}
                <br />
                If a new version shipped, reloading picks it up.
              </>
            )}
            {!isCrash && (
              <>
                $ curl {pathname}
                <br />
                <span className="text-destructive">404</span> No page at this address.
              </>
            )}
          </p>
          <div className="mt-7 flex w-full flex-col gap-2 sm:w-fit sm:flex-row">
            {isCrash && (
              <Button variant="outline" onClick={() => window.location.reload()}>
                Reload page
              </Button>
            )}
            {!isCrash && (
              <Button asChild>
                <Link to="/">Go home</Link>
              </Button>
            )}
            {!isCrash && (
              <Button variant="outline" onClick={() => navigate(-1)}>
                Go back
              </Button>
            )}
          </div>
        </div>
      </div>
    </ShowcaseSurface>
  );
};
