import type { ReactNode } from "react";
import { View } from "react-native";

import { Text } from "@/components/ui/text";

interface StackSecretsChangedNoticeProps {
  names: string[];
  // The Redeploy action, for a viewer who may use it.
  children?: ReactNode;
}

// Says which referenced secrets changed value since the last deploy; the next deploy clears it.
export const StackSecretsChangedNotice = ({ names, children }: StackSecretsChangedNoticeProps) => (
  <View className="gap-3 rounded-xl border border-border bg-surface-2 p-4">
    <View className="flex-row items-start gap-2.5">
      <View className="mt-1.5 size-2.5 rounded-full bg-warning-mark" />
      <Text className="flex-1 font-semibold">{`${names.join(", ")} changed since the last deploy`}</Text>
    </View>
    {children}
  </View>
);
