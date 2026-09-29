import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { providerLabel, type Identity } from "@/models/User";

interface SignInAccountRowProps {
  identity: Identity;
}

export const SignInAccountRow = ({ identity }: SignInAccountRowProps) => (
  <View className="min-h-11 flex-row items-center justify-between gap-2 border-b border-border bg-card px-3 py-3">
    <Text className="font-medium">{providerLabel[identity.provider] ?? identity.provider}</Text>
    <Text variant="muted" numberOfLines={1} className="text-xs">
      {identity.login || identity.name}
    </Text>
  </View>
);
