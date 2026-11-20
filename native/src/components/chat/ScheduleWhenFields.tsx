import DateTimePicker, { DateTimePickerAndroid } from "@react-native-community/datetimepicker";
import { Platform, Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { formatDayLabel, formatMessageTime } from "@/lib/time";

interface ScheduleWhenFieldsProps {
  at: Date;
  onChange: (at: Date) => void;
}

// The day and the time it posts, in the phone's own zone: Android opens its date and time dialogs from two rows, iOS
// shows its compact picker in place.
export const ScheduleWhenFields = ({ at, onChange }: ScheduleWhenFieldsProps) => {
  const pick = (mode: "date" | "time") =>
    DateTimePickerAndroid.open({ value: at, mode, is24Hour: true, minimumDate: new Date(), onValueChange: (_e, date) => onChange(date) });
  return (
    <View className="flex-row items-center justify-between gap-3">
      {Platform.OS === "ios" && <Text className="text-row font-medium">When</Text>}
      {Platform.OS === "ios" && (
        <DateTimePicker value={at} mode="datetime" display="compact" minimumDate={new Date()} onValueChange={(_e, date) => onChange(date)} />
      )}
      {Platform.OS !== "ios" && <WhenRow label="Day" value={formatDayLabel(at.toISOString())} onPress={() => pick("date")} />}
      {Platform.OS !== "ios" && <WhenRow label="Time" value={formatMessageTime(at.toISOString())} onPress={() => pick("time")} />}
    </View>
  );
};

const WhenRow = ({ label, value, onPress }: { label: string; value: string; onPress: () => void }) => (
  <Pressable
    role="button"
    aria-label={`${label}, ${value}`}
    onPress={onPress}
    className="min-h-12 flex-1 justify-center gap-0.5 rounded-md border border-input bg-card px-3 py-2 active:bg-accent"
  >
    <Text className="text-micro text-muted-foreground">{label}</Text>
    <Text className="text-body">{value}</Text>
  </Pressable>
);
