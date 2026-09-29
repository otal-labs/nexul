import { View } from "react-native";

import { errorMessage, isNotFound } from "@/api/client";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ErrorDisplayProps {
  error: unknown;
  // A detail screen passes what its 404 means; the raw server message is for every other failure.
  notFound?: string;
  className?: string;
}

export const ErrorDisplay = ({ error, notFound, className }: ErrorDisplayProps) => {
  const missing = notFound !== undefined && isNotFound(error);
  return (
    <View accessible role="alert" className={cn("flex-row items-start gap-2 px-4 py-2", className)}>
      {!missing && <View className="mt-1.5 size-2 rounded-full bg-destructive" />}
      <Text variant="small" className={cn("flex-1 leading-5", missing ? "text-muted-foreground" : "text-foreground")}>
        {missing ? notFound : errorMessage(error)}
      </Text>
    </View>
  );
};
