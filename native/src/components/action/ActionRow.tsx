import Lock from "lucide-react-native/icons/lock";
import type { ReactNode } from "react";
import { Pressable, View } from "react-native";

import type { ActionGate } from "@nexul/client-core/actionState";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ActionRowProps extends ActionGate {
  label: string;
  onPress: () => void;
  leading?: ReactNode;
  destructive?: boolean;
}

export const ActionRow = ({ permission, projectId, answer, verb, label, onPress, leading, destructive }: ActionRowProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && (
        <Pressable role="button" onPress={onPress} className="min-h-[52px] flex-row items-center gap-3 px-5 py-2.5 active:bg-accent">
          {leading}
          <Text className={cn("flex-1 text-[15px]", destructive && "text-destructive")}>{label}</Text>
        </Pressable>
      )}
      {action.state === "locked" && (
        <View role="button" aria-label={label} aria-disabled accessibilityHint={action.reason} accessible className="min-h-[52px] flex-row items-start gap-3 px-5 py-2.5">
          {leading}
          <View className="flex-1 gap-0.5">
            <Text className="text-[15px] text-muted-foreground">{label}</Text>
            <Text aria-hidden className="text-[13px] text-muted-foreground">{action.reason}</Text>
          </View>
          <Icon as={Lock} className="mt-1 size-4 text-muted-foreground" />
        </View>
      )}
    </>
  );
};
