import type { ReactNode } from "react";
import { View } from "react-native";

import type { Area } from "@nexul/client-core/permissions";

import { EmptyState } from "@/components/EmptyState";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";

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
      {allowed === false && (
        <View className="flex-1 bg-background">
          <EmptyState title="This page doesn't exist" message="It may have moved, or your role doesn't include it." />
        </View>
      )}
      {allowed === true && children}
    </>
  );
};
