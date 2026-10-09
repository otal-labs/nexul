import type { ReactNode } from "react";
import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { MIN_SERVER_VERSION } from "@/lib/serverVersion";

interface ServerRefusedScreenProps {
  host: string;
  version: string;
  retrying: boolean;
  onRetry: () => void;
  // The way out besides waiting for an upgrade: Sign out when signed in, another server when not.
  children: ReactNode;
}

export const ServerRefusedScreen = ({ host, version, retrying, onRetry, children }: ServerRefusedScreenProps) => (
  <View className="flex-1 justify-center gap-6 bg-background px-6">
    <Text role="heading" className="font-display text-[28px] leading-[32px] tracking-[-0.5px]">
      This server needs an upgrade
    </Text>
    <Text className="text-base leading-6 text-muted-foreground">
      <Text className="font-mono">{host}</Text> runs <Text className="font-mono">{version}</Text>. This app needs{" "}
      <Text className="font-mono">{MIN_SERVER_VERSION}</Text> or newer. Ask whoever runs it to upgrade.
    </Text>
    <Button variant="outline" disabled={retrying} onPress={onRetry}>
      <Text>Retry</Text>
    </Button>
    {children}
  </View>
);
