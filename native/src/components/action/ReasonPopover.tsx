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

export const ReasonPopover = ({ name, reason, anchor, onClose }: ReasonPopoverProps) => {
  const { width } = useWindowDimensions();
  return (
    <Modal visible={anchor !== null} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable role="button" aria-label="Close" className="flex-1" onPress={onClose}>
        {anchor && (
          <View
            accessibilityViewIsModal
            className="absolute w-72 max-w-[85%] gap-0.5 rounded-xl border border-border bg-popover p-3.5"
            style={{ top: anchor.y + anchor.height + 8, right: Math.max(16, width - anchor.x - anchor.width) }}
          >
            <Text className="text-[15px] font-medium">{name} is locked</Text>
            <Text className="text-[13px] text-muted-foreground">{reason}</Text>
          </View>
        )}
      </Pressable>
    </Modal>
  );
};
