import { AlertCircle, ExternalLink } from "lucide-react";

import { errorMessage } from "@/api/client";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { parseInstallUrl } from "@/models/Repository";

// A scan that failed: the GitHub App missing on the repository gets its install link, any other failure its message.
export const RepositoryScanError = ({ error }: { error: unknown }) => {
  const notInstalled = (error as { response?: { status?: number } })?.response?.status === 404;
  const installUrl = notInstalled ? parseInstallUrl(errorMessage(error)) : undefined;

  return (
    <>
      {notInstalled && (
        <EmptyState
          icon={AlertCircle}
          title="The GitHub App isn't installed on this repository"
          message={errorMessage(error)}
          size="compact"
          action={
            installUrl && (
              <Button asChild variant="outline" size="sm">
                <a href={installUrl} target="_blank" rel="noreferrer">
                  Install the GitHub App
                  <ExternalLink className="size-3.5" aria-hidden />
                </a>
              </Button>
            )
          }
        />
      )}
      {!notInstalled && <ErrorDisplay error={error} title="Couldn't scan the repository." />}
    </>
  );
};
