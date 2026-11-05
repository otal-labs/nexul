import { type ReactNode } from "react";
import { useMatches } from "react-router";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ErrorScreen } from "@/components/ErrorScreen";
import { useCanOpen } from "@/hooks/AccessHooks";
import { usePageChunk } from "@/hooks/usePageChunk";
import type { RouteAccess } from "@/models/Access";

interface AreaGateProps {
  children: ReactNode;
}

// The one route-level permission check: a route whose handle names an area the viewer can't open is not found, unloaded.
export const AreaGate = ({ children }: AreaGateProps) => {
  const match = useMatches().at(-1);
  const handle = match?.handle as RouteAccess | undefined;
  const area = handle?.area;
  const canOpen = useCanOpen();
  const allowed = area === undefined || canOpen(area);
  const pageReady = usePageChunk(match?.id ?? "", handle?.page, allowed === true);

  return (
    <>
      {(allowed === undefined || (allowed && !pageReady)) && <LoadingDisplay />}
      {allowed === false && <ErrorScreen />}
      {allowed === true && pageReady && children}
    </>
  );
};
