import { Modal, Pressable, useWindowDimensions, View } from "react-native";

import { Text } from "@/components/ui/text";

export interface Anchor {
  x: number;
  y: number;
  width: number;
  height: number;
}

interface ReasonPopoverProps {
  name: string;
  reason: string;
  anchor: Anchor | null;
  onClose: () => void;
}

const POPOVER_WIDTH = 288;
const MARGIN = 16;

// Starts at the control's left edge and slides left only as far as the window needs, so a control at either edge keeps it on screen.
export const ReasonPopover = ({ name, reason, anchor, onClose }: ReasonPopoverProps) => {
  const { width: windowWidth } = useWindowDimensions();
  const width = Math.min(POPOVER_WIDTH, windowWidth - 2 * MARGIN);
  return (
    <Modal visible={anchor !== null} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable role="button" aria-label="Close" className="flex-1" onPress={onClose}>
        {anchor && (
          <View
            accessibilityViewIsModal
            className="absolute gap-0.5 rounded-xl border border-border bg-popover p-3.5"
            style={{ top: anchor.y + anchor.height + 8, width, left: Math.min(Math.max(MARGIN, anchor.x), windowWidth - MARGIN - width) }}
          >
            <Text className="text-[15px] font-medium">{name} is locked</Text>
            <Text className="text-[13px] text-muted-foreground">{reason}</Text>
          </View>
        )}
      </Pressable>
    </Modal>
  );
};
