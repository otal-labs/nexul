import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { DocView } from "@/hooks/useDocView";

interface DocViewSwitchProps {
  view: DocView;
  waiting: number;
  onChange: (view: DocView) => void;
}

// The "Doc | Questions N" switch under the title; N is the questions waiting on an answer, and the only marker on the doc.
export const DocViewSwitch = ({ view, waiting, onChange }: DocViewSwitchProps) => (
  <View className="px-5 pb-4">
    <ToggleGroup
      type="single"
      variant="outline"
      value={view}
      // Pressing the active item reports an empty value; the current view stays.
      onValueChange={(value) => value && onChange(value as DocView)}
      aria-label="Doc or questions"
      className="w-full"
    >
      <ToggleGroupItem value="doc" isFirst className="h-12 flex-1" aria-label="Doc">
        <Text>Doc</Text>
      </ToggleGroupItem>
      <ToggleGroupItem
        value="questions"
        isLast
        className="h-12 flex-1"
        aria-label={waiting > 0 ? `Questions, ${waiting} waiting` : "Questions"}
      >
        <Text>Questions</Text>
        {waiting > 0 && <Text className="font-mono text-xs leading-snug text-muted-foreground">{waiting}</Text>}
      </ToggleGroupItem>
    </ToggleGroup>
  </View>
);
