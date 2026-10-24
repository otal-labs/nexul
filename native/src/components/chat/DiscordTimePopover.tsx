import { Modal, Pressable, useWindowDimensions, View } from "react-native";

import { fullDiscordTime, type DiscordTime } from "@nexul/client-core/discordTime";

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

// Under the finger when the tap says where it landed, else a third of the way down; the code is selectable, so a hold copies it.
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
            <Text className="text-[13px] text-muted-foreground">Hold the code to copy it.</Text>
          </View>
        )}
      </Pressable>
    </Modal>
  );
};
