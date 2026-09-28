import { useRouter } from "expo-router";
import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";

export default function MoreScreen() {
  const router = useRouter();
  return (
    <View className="flex-1 bg-background">
      <Pressable
        role="button"
        onPress={() => router.push("/more/docs")}
        className="min-h-11 flex-row items-center border-b border-border px-4 py-3 active:bg-accent"
      >
        <Text>Docs</Text>
      </Pressable>
      <Text variant="muted" className="px-4 py-3">
        Runners and Your settings land here.
      </Text>
    </View>
  );
}
