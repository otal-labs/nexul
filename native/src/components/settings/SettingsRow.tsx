import { Pressable } from "react-native";

import { Text } from "@/components/ui/text";

interface SettingsRowProps {
  label: string;
  meta?: string;
  onPress: () => void;
}

export const SettingsRow = ({ label, meta, onPress }: SettingsRowProps) => (
  <Pressable
    role="button"
    onPress={onPress}
    className="min-h-11 flex-row items-center justify-between gap-2 border-b border-border bg-card px-4 py-3 active:bg-accent"
  >
    <Text className="font-medium">{label}</Text>
    {meta && (
      <Text variant="muted" numberOfLines={1} className="font-mono text-xs">
        {meta}
      </Text>
    )}
  </Pressable>
);
