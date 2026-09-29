import { type ReactNode } from "react";
import { useMatches } from "react-router";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ErrorScreen } from "@/components/ErrorScreen";
import { useCanOpen } from "@/hooks/AccessHooks";
import type { RouteAccess } from "@/models/Access";

interface AreaGateProps {
  children: ReactNode;
}

// The one route-level permission check: a route whose handle names an area the viewer can't open is not found.
export const AreaGate = ({ children }: AreaGateProps) => {
  const matches = useMatches();
  const area = (matches.at(-1)?.handle as RouteAccess | undefined)?.area;
  const canOpen = useCanOpen();
  const allowed = area === undefined || canOpen(area);

  return (
    <>
      {allowed === undefined && <LoadingDisplay />}
      {allowed === false && <ErrorScreen />}
      {allowed === true && children}
    </>
  );
};
