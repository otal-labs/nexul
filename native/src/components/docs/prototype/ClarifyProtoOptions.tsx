import { Pressable, View } from "react-native";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Text } from "@/components/ui/text";
import { optionValue, type AnswerValue, type ProtoOption, type ProtoQuestion } from "@/components/docs/prototype/ClarifyProtoData";
import { cn } from "@/lib/utils";

interface ClarifyProtoOptionsProps {
  item: ProtoQuestion;
  draft: AnswerValue | undefined;
  onDraft: (value: AnswerValue) => void;
}

const toggle = (item: ProtoQuestion, draft: AnswerValue | undefined, value: string): AnswerValue => {
  if (!item.multi_select) return { selected: [value] };
  const current = draft?.selected ?? [];
  return { selected: current.includes(value) ? current.filter((v) => v !== value) : [...current, value] };
};

interface OptionRowProps {
  option: ProtoOption;
  first: boolean;
  checked: boolean;
  multi: boolean;
  onPress: () => void;
}

// One option as a whole-row target: the radio or checkbox, the label, the description muted under it.
const OptionRow = ({ option, first, checked, multi, onPress }: OptionRowProps) => (
  <Pressable
    role={multi ? "checkbox" : "radio"}
    aria-checked={checked}
    onPress={onPress}
    android_ripple={{ borderless: false }}
    className={cn("min-h-12 flex-row items-start gap-3 px-3 py-3", !first && "border-t border-border", checked && "bg-accent")}
  >
    <View pointerEvents="none" importantForAccessibility="no-hide-descendants" className="mt-0.5">
      {multi && <Checkbox checked={checked} onCheckedChange={() => {}} className="size-5" />}
      {!multi && <RadioGroupItem value={optionValue(option)} aria-label={option.label} className="size-5" />}
    </View>
    <View className="min-w-0 flex-1">
      <Text className="leading-snug">{option.label}</Text>
      {option.description && <Text className="mt-0.5 text-sm text-muted-foreground leading-snug">{option.description}</Text>}
    </View>
  </Pressable>
);

// The answering part of one question: the hint, the option rows, and "Something else…" for the client's own words.
export const ClarifyProtoOptions = ({ item, draft, onDraft }: ClarifyProtoOptionsProps) => {
  const selected = draft?.selected ?? [];
  const hasOptions = item.options.length > 0;
  const rows = item.options.map((option, i) => (
    <OptionRow
      key={optionValue(option)}
      option={option}
      first={i === 0}
      checked={selected.includes(optionValue(option))}
      multi={!!item.multi_select}
      onPress={() => onDraft(toggle(item, draft, optionValue(option)))}
    />
  ));
  return (
    <View>
      {item.header && <Text className="text-sm text-muted-foreground leading-snug">{item.header}</Text>}
      {hasOptions && (
        <View className="mt-2 overflow-hidden rounded-md border border-border">
          {item.multi_select && rows}
          {!item.multi_select && <RadioGroup value={selected[0] ?? ""} onValueChange={() => {}} className="gap-0">{rows}</RadioGroup>}
        </View>
      )}
      <Input
        aria-label={hasOptions ? "Something else" : "Your answer"}
        placeholder={hasOptions ? "Something else…" : "Your answer"}
        value={draft?.text ?? ""}
        onChangeText={(text) => onDraft({ text })}
        className="mt-2 h-12"
      />
    </View>
  );
};
