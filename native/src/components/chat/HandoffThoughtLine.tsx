import Brain from "lucide-react-native/icons/brain";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";

interface HandoffThoughtLineProps {
  text: string;
}

// A thought the helper summarised between its steps, muted and in italics so it never reads as something it said.
export const HandoffThoughtLine = ({ text }: HandoffThoughtLineProps) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <View accessible aria-label={`Thought: ${text}`} className="flex-row items-start gap-2.5 px-1">
      <View className="h-5 justify-center">
        <Brain color={String(muted)} size={14} />
      </View>
      <Text className="min-w-0 flex-1 text-sm italic text-muted-foreground">{text}</Text>
    </View>
  );
};
