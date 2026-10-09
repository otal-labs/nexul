import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { providerLabel, type Identity } from "@/models/User";

interface SignInAccountRowProps {
  identity: Identity;
  first: boolean;
}

export const SignInAccountRow = ({ identity, first }: SignInAccountRowProps) => (
  <View className={cn("min-h-12 flex-row items-center justify-between gap-2 px-4 py-3", !first && "border-t border-border")}>
    <Text className="text-[15px]">{providerLabel[identity.provider] ?? identity.provider}</Text>
    <Text numberOfLines={1} className="font-mono text-xs text-muted-foreground">
      {identity.login || identity.name}
    </Text>
  </View>
);
