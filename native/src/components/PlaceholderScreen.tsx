import { View } from "react-native";

import { Text } from "@/components/ui/text";

interface PlaceholderScreenProps {
  message: string;
}

export const PlaceholderScreen = ({ message }: PlaceholderScreenProps) => (
  <View className="flex-1 items-center justify-center bg-background px-6">
    <Text variant="muted" className="text-center">
      {message}
    </Text>
  </View>
);
