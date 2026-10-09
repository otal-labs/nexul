import { createContext, useContext, useMemo, useState } from "react";
import { Gesture, type PanGesture } from "react-native-gesture-handler";
import { measure, useAnimatedRef, useSharedValue, type AnimatedRef, type SharedValue } from "react-native-reanimated";
import { scheduleOnRN } from "react-native-worklets";
import type { View } from "react-native";

import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Lifted {
  ticket: Ticket;
  // Where the card sat on screen when it was picked up.
  from: Rect;
  // Set once the finger lets go: the drop target's index, or -1 to go back.
  landing?: number;
}

export interface BoardDrag {
  lifted: Lifted | null;
  lift: (ticket: Ticket, from: Rect) => void;
  release: (target: number) => void;
  clear: () => void;
  // The finger's travel since the pick-up and the drop target under it (-1 for none), on the UI thread.
  tx: SharedValue<number>;
  ty: SharedValue<number>;
  hovered: SharedValue<number>;
  // Where the finger took hold of the card, in the card's own coordinates.
  grab: SharedValue<{ x: number; y: number }>;
  targets: SharedValue<Rect[]>;
  // The board's columns, which are the drop targets in order, and the move a drop or an accessibility action makes.
  statuses: BoardStatus[];
  move: (ticketId: string, statusId: string) => void;
}

export const BoardDragContext = createContext<BoardDrag | null>(null);

export const useBoardDragState = (statuses: BoardStatus[], move: BoardDrag["move"]): BoardDrag => {
  const [lifted, setLifted] = useState<Lifted | null>(null);
  const tx = useSharedValue(0);
  const ty = useSharedValue(0);
  const hovered = useSharedValue(-1);
  const grab = useSharedValue({ x: 0, y: 0 });
  const targets = useSharedValue<Rect[]>([]);
  // Stable, so a card's gesture is not rebuilt mid-drag when the board re-renders around the lifted card.
  const [actions] = useState(() => ({
    lift: (ticket: Ticket, from: Rect) => setLifted({ ticket, from }),
    release: (target: number) => setLifted((current) => current && { ...current, landing: target }),
    clear: () => setLifted(null),
  }));
  return {
    lifted,
    ...actions,
    tx,
    ty,
    hovered,
    grab,
    targets,
    statuses,
    move,
  };
};

const hitTest = (rects: Rect[], x: number, y: number) => {
  "worklet";
  return rects.findIndex((r) => x >= r.x && x <= r.x + r.width && y >= r.y && y <= r.y + r.height);
};

// A held card follows the finger after a long press, so a plain swipe still scrolls the board.
export const useCardDrag = (ticket: Ticket): { ref: AnimatedRef<View>; gesture: PanGesture | undefined } => {
  const ref = useAnimatedRef<View>();
  const drag = useContext(BoardDragContext);
  const armed = useSharedValue(false);
  // Only the stable parts of the drag go in, so the gesture is not rebuilt while the board re-renders mid-drag.
  const { tx, ty, hovered, grab, targets, lift, release } = drag ?? {};
  const gesture = useMemo(() => {
    if (!tx || !ty || !hovered || !grab || !targets || !lift || !release) return undefined;
    return Gesture.Pan()
      .activateAfterLongPress(280)
      .onStart((e) => {
        const box = measure(ref);
        if (!box) return;
        armed.set(true);
        tx.set(0);
        ty.set(0);
        hovered.set(-1);
        grab.set({ x: e.x, y: e.y });
        scheduleOnRN(lift, ticket, { x: box.pageX, y: box.pageY, width: box.width, height: box.height });
      })
      .onUpdate((e) => {
        tx.set(e.translationX);
        ty.set(e.translationY);
        hovered.set(hitTest(targets.get(), e.absoluteX, e.absoluteY));
      })
      .onFinalize(() => {
        if (!armed.get()) return;
        armed.set(false);
        scheduleOnRN(release, hovered.get());
      });
  }, [ticket, ref, armed, tx, ty, hovered, grab, targets, lift, release]);
  return { ref, gesture };
};
