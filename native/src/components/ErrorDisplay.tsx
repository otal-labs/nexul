import { View } from "react-native";

import { errorMessage } from "@/api/client";
import { Text } from "@/components/ui/text";

interface ErrorDisplayProps {
  error: unknown;
}

export const ErrorDisplay = ({ error }: ErrorDisplayProps) => (
  <View accessible role="alert" className="flex-row items-start gap-2 py-2">
    <View className="mt-1.5 size-2 rounded-full bg-destructive" />
    <Text variant="small" className="flex-1 leading-5 text-foreground">
      {errorMessage(error)}
    </Text>
  </View>
);
