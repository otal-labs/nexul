import { optionValue, toggleOption, type AnswerValue, type QuestionItem, type QuestionOption } from "@nexul/client-core/question";
import { Pressable, View } from "react-native";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ClarificationOptionsProps {
  item: QuestionItem;
  draft: AnswerValue | undefined;
  onDraft: (value: AnswerValue) => void;
}

interface OptionRowProps {
  option: QuestionOption;
  first: boolean;
  checked: boolean;
  multi: boolean;
  onPress: () => void;
}

// A whole-row target: the radio or checkbox shows the state, the row takes the touch and names it to a screen reader.
const OptionRow = ({ option, first, checked, multi, onPress }: OptionRowProps) => (
  <Pressable
    role={multi ? "checkbox" : "radio"}
    aria-checked={checked}
    aria-label={option.label}
    onPress={onPress}
    className={cn("min-h-12 flex-row items-start gap-3 px-3 py-3 active:bg-accent", !first && "border-t border-border", checked && "bg-accent")}
  >
    <View pointerEvents="none" importantForAccessibility="no-hide-descendants" className="mt-0.5">
      {multi && <Checkbox checked={checked} onCheckedChange={() => {}} className="size-5" />}
      {!multi && <RadioGroupItem value={optionValue(option)} className="size-5" />}
    </View>
    <View className="min-w-0 flex-1">
      <Text className="leading-snug">{option.label}</Text>
      {option.description && <Text className="mt-0.5 text-sm leading-snug text-muted-foreground">{option.description}</Text>}
    </View>
  </Pressable>
);

// The answering part of one question: the option rows, and "Something else…" for the person's own words.
export const ClarificationOptions = ({ item, draft, onDraft }: ClarificationOptionsProps) => {
  const selected = draft?.selected ?? [];
  const hasOptions = item.options.length > 0;
  const multi = !!item.multi_select;
  const rows = item.options.map((option, i) => (
    <OptionRow
      key={optionValue(option)}
      option={option}
      first={i === 0}
      checked={selected.includes(optionValue(option))}
      multi={multi}
      onPress={() => onDraft(toggleOption(item, draft, optionValue(option)))}
    />
  ));
  return (
    <View>
      {hasOptions && (
        <View className="mt-2 overflow-hidden rounded-md border border-border">
          {multi && rows}
          {!multi && (
            <RadioGroup value={selected[0] ?? ""} onValueChange={() => {}} className="gap-0">
              {rows}
            </RadioGroup>
          )}
        </View>
      )}
      <Input
        aria-label={hasOptions ? "Something else" : "Your answer"}
        placeholder={hasOptions ? "Something else…" : "Your answer"}
        value={draft?.text ?? ""}
        onChangeText={(text) => onDraft({ text })}
        className="mt-2 h-auto min-h-12"
      />
    </View>
  );
};
