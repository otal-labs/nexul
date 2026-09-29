import type { AxiosError } from "axios";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ErrorScreen } from "@/components/ErrorScreen";

// The server answers 403 or 404 for an item the viewer can't see, and a deep link to either is simply not found.
const isMissing = (error: unknown) => [403, 404].includes((error as AxiosError | undefined)?.response?.status ?? 0);

interface DetailErrorDisplayProps {
  error: unknown;
  // Embedded in another page (Inbox's split view): a missing item is an error box there, not a whole not-found page.
  embedded?: boolean;
}

// A detail page's own query error: the app's not-found page for a missing item, the error box for anything else.
export const DetailErrorDisplay = ({ error, embedded = false }: DetailErrorDisplayProps) => {
  const notFound = !embedded && isMissing(error);
  return (
    <>
      {notFound && <ErrorScreen />}
      {!notFound && <ErrorDisplay error={error} />}
    </>
  );
};
