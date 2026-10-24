import { setStringAsync } from "expo-clipboard";
import { createEventInCalendarAsync } from "expo-calendar/legacy";
import CalendarPlus from "lucide-react-native/icons/calendar-plus";
import Check from "lucide-react-native/icons/check";
import Copy from "lucide-react-native/icons/copy";
import { useContext, useState } from "react";
import { Modal, Pressable, useWindowDimensions, View } from "react-native";

import { fullDiscordTime, type DiscordTime } from "@nexul/client-core/discordTime";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";

export interface PressPoint {
  x: number;
  y: number;
}

export interface OpenTime {
  time: DiscordTime;
  at: PressPoint | null;
}

interface DiscordTimePopoverProps {
  open: OpenTime | null;
  onClose: () => void;
}

const POPOVER_WIDTH = 288;
const MARGIN = 16;
const HOUR_MS = 60 * 60 * 1000;

// The system's own new-event screen, which needs no calendar permission, holding a one-hour event at the instant.
const addToCalendar = (date: Date, title: string, onDone: () => void) =>
  createEventInCalendarAsync({ title, startDate: date, endDate: new Date(date.getTime() + HOUR_MS) })
    .catch((error: unknown) => console.warn("time code: calendar dialog failed", error))
    .finally(onDone);

const TimeActions = ({ time, onClose }: { time: DiscordTime; onClose: () => void }) => {
  const title = useContext(CalendarTitleContext);
  const [copied, setCopied] = useState(false);
  const copy = () =>
    setStringAsync(time.code).then(
      () => setCopied(true),
      (error: unknown) => console.warn("time code: copy failed", error),
    );
  return (
    <View className="gap-2 pt-1.5">
      <Button variant="outline" size="sm" onPress={copy}>
        <Icon as={copied ? Check : Copy} size={14} className={copied ? "text-success" : "text-foreground"} />
        <Text>{copied ? "Copied" : "Copy code"}</Text>
      </Button>
      <Button variant="outline" size="sm" onPress={() => addToCalendar(time.date, title, onClose)}>
        <Icon as={CalendarPlus} size={14} className="text-foreground" />
        <Text>Add to calendar</Text>
      </Button>
    </View>
  );
};

// Under the finger when the tap says where it landed, else a third of the way down.
export const DiscordTimePopover = ({ open, onClose }: DiscordTimePopoverProps) => {
  const { width: windowWidth, height: windowHeight } = useWindowDimensions();
  const width = Math.min(POPOVER_WIDTH, windowWidth - 2 * MARGIN);
  const at = open?.at;
  const left = at ? Math.min(Math.max(MARGIN, at.x - width / 2), windowWidth - MARGIN - width) : (windowWidth - width) / 2;
  const top = at ? at.y + 16 : windowHeight / 3;
  return (
    <Modal visible={open !== null} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable role="button" aria-label="Close" className="flex-1" onPress={onClose}>
        {open && (
          <View
            accessibilityViewIsModal
            onStartShouldSetResponder={() => true}
            className="absolute gap-1.5 rounded-xl border border-border bg-popover p-3.5"
            style={{ top, left, width }}
          >
            <Text className="text-[15px] font-medium">{fullDiscordTime(open.time.date)}</Text>
            <Text selectable className="font-mono text-[13px]">
              {open.time.code}
            </Text>
            <TimeActions time={open.time} onClose={onClose} />
          </View>
        )}
      </Pressable>
    </Modal>
  );
};
