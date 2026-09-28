import { ActivityIndicator, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";

interface LoadingDisplayProps {
  message?: string;
}

export const LoadingDisplay = ({ message }: LoadingDisplayProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <View accessible role="progressbar" className="flex-1 items-center justify-center gap-3 px-6 py-8">
      <ActivityIndicator color={String(mutedForeground)} />
      {message && <Text variant="muted">{message}</Text>}
    </View>
  );
};
