import type { ReactNode } from "react";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import type { Area } from "@/models/Access";

interface AreaGateProps {
  area: Area;
  children: ReactNode;
}

// A deep link into an area the viewer can't read is plainly not found, the same as a missing item.
export const AreaGate = ({ area, children }: AreaGateProps) => {
  const allowed = useAreaAccess()?.(area);
  return (
    <>
      {allowed === undefined && <LoadingDisplay />}
      {allowed === false && <PlaceholderScreen message="This page doesn't exist." />}
      {allowed === true && children}
    </>
  );
};
