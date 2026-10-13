import type { ReactNode } from "react";
import { useMatches } from "react-router";

import { MemberWaiting } from "@/components/accessgrid/MemberWaiting";
import { useAccessSetupState } from "@/hooks/AccessGridHooks";
import type { RouteAccess } from "@/models/Access";

interface AccessSetupGateProps {
  children: ReactNode;
}

// A page that shows a project waits, for a member no project lists, until an Owner finishes the one-time setup.
export const AccessSetupGate = ({ children }: AccessSetupGateProps) => {
  const matches = useMatches();
  const setup = useAccessSetupState();
  const waiting = setup === "member" && (matches.at(-1)?.handle as RouteAccess | undefined)?.needsProject === true;
  return (
    <>
      {waiting && <MemberWaiting />}
      {!waiting && children}
    </>
  );
};
