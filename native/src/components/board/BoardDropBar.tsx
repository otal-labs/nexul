import { useContext, useRef } from "react";
import { View, type LayoutChangeEvent } from "react-native";
import Animated, { FadeIn, useAnimatedStyle } from "react-native-reanimated";
import { useCSSVariable } from "uniwind";

import { BoardDragContext, type Rect } from "@/components/board/boardDrag";
import { Microheader } from "@/components/Microheader";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { statusStageDot, type BoardStatus } from "@/models/Status";

const DropTarget = ({ status, index, current }: { status: BoardStatus; index: number; current: boolean }) => {
  const drag = useContext(BoardDragContext);
  const [brand] = useCSSVariable(["--color-brand"]);
  const hovered = drag?.hovered;
  const edge = String(brand);
  // The column under the finger is the selection, so it takes the ember edge and grows a little toward the finger.
  const style = useAnimatedStyle(() => {
    const on = hovered?.get() === index;
    return { borderColor: on ? edge : "transparent", transform: [{ scale: on ? 1.04 : 1 }] };
  });
  return (
    <Animated.View style={style} className={cn("min-h-12 flex-row items-center gap-2 rounded-lg border-2 bg-card px-3.5", current && "opacity-50")}>
      <View className={cn("size-2 rounded-full", statusStageDot(status.kind))} />
      <Text className="text-sm font-medium">{status.name}</Text>
    </Animated.View>
  );
};

// The columns as drop targets along the bottom while a card is held; it fades in with the pick-up, so the targets are
// measured where they rest.
export const BoardDropBar = () => {
  const drag = useContext(BoardDragContext);
  const rects = useRef<Rect[]>([]);
  if (!drag) return null;
  const { statuses, targets, lifted } = drag;

  const onChip = (i: number) => (e: LayoutChangeEvent) => {
    e.target.measureInWindow((x, y, width, height) => {
      rects.current[i] = { x, y, width, height };
      targets.set([...rects.current]);
    });
  };

  return (
    <Animated.View
      entering={FadeIn.duration(150)}
      className="absolute inset-x-0 bottom-0 gap-2.5 overflow-hidden rounded-t-xl border-t border-border bg-popover px-4 pb-4 pt-3"
    >
      <Microheader>Move to</Microheader>
      <View className="flex-row flex-wrap gap-2">
        {statuses.map((status, i) => (
          <View key={status.id} onLayout={onChip(i)}>
            <DropTarget status={status} index={i} current={status.id === lifted?.ticket.status} />
          </View>
        ))}
      </View>
    </Animated.View>
  );
};
