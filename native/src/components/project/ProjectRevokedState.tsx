import { useRouter } from "expo-router";
import FolderLock from "lucide-react-native/icons/folder-lock";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

// What a screen shows once its project was taken away while it was open (ADR 0097).
export const ProjectRevokedState = () => {
  const router = useRouter();
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <View className="flex-1 items-center justify-center gap-3 bg-background px-6">
      <View className="size-10 items-center justify-center rounded-md bg-muted/60">
        <FolderLock size={20} color={String(mutedForeground)} />
      </View>
      <Text className="text-center font-semibold">You no longer have access to this project</Text>
      <Text variant="muted" className="text-center">
        Someone changed your access. Projects you can still open are in the project picker.
      </Text>
      <Button onPress={() => router.navigate("/inbox")}>
        <Text>Go to Inbox</Text>
      </Button>
    </View>
  );
};
