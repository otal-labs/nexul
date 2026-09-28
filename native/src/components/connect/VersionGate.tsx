import type { ReactNode } from "react";
import { View } from "react-native";

import { ServerRefusedScreen } from "@/components/connect/ServerRefusedScreen";
import { useFetchAbout } from "@/hooks/ConnectHooks";
import { serverIsSupported } from "@/lib/serverVersion";
import { useSessionStore } from "@/stores/sessionStore";

interface VersionGateProps {
  children: ReactNode;
}

// A signed-in phone behind a refusal keeps its session; only the screens are held back until the server is upgraded.
export const VersionGate = ({ children }: VersionGateProps) => {
  const host = useSessionStore((s) => (s.signedIn ? s.host : null));
  const { data, isFetching, refetch } = useFetchAbout(host);
  const refusedVersion = data && !serverIsSupported(data.version) ? data.version : undefined;

  return (
    <View className="flex-1">
      {host && refusedVersion && (
        <ServerRefusedScreen
          host={host}
          version={refusedVersion}
          retrying={isFetching}
          onRetry={() => void refetch()}
        />
      )}
      {!refusedVersion && children}
    </View>
  );
};
