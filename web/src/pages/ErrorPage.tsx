import { useRouteError } from "react-router";

import { ErrorScreen } from "@/components/ErrorScreen";

// Doubles as the "*" not-found route and the errorElement; useRouteError() returning non-null tells them apart.
export const ErrorPage = () => <ErrorScreen error={useRouteError()} />;
