import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { MIN_SERVER_VERSION } from "@/lib/serverVersion";

interface ServerRefusedScreenProps {
  host: string;
  version: string;
  retrying: boolean;
  onRetry: () => void;
}

export const ServerRefusedScreen = ({ host, version, retrying, onRetry }: ServerRefusedScreenProps) => (
  <View className="flex-1 justify-center gap-6 bg-background px-6">
    <Text className="leading-6">
      <Text className="font-mono">{host}</Text> runs <Text className="font-mono">{version}</Text>. This app needs{" "}
      <Text className="font-mono">{MIN_SERVER_VERSION}</Text> or newer. Ask whoever runs it to upgrade.
    </Text>
    <Button variant="outline" disabled={retrying} onPress={onRetry}>
      <Text>Retry</Text>
    </Button>
  </View>
);
