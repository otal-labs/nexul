import { useContext, useEffect, useRef } from "react";
import { StyleSheet, View } from "react-native";
import Animated, { measure, useAnimatedRef, useAnimatedStyle, useDerivedValue, useSharedValue, withSpring, withTiming } from "react-native-reanimated";
import { runOnUI, scheduleOnRN } from "react-native-worklets";

import { BoardDragContext, useLifted } from "@/components/board/boardDrag";
import { BoardDropBar } from "@/components/board/BoardDropBar";
import { TicketCardBody, ticketCardClass } from "@/components/board/TicketCard";
import type { TicketType } from "@/hooks/TicketTypeHooks";
import { ease, useReducedMotion } from "@/lib/motion";
import { StatusKind } from "@/models/Status";

interface BoardDragLayerProps {
  projectPrefix: string | undefined;
  ticketTypes: TicketType[] | undefined;
}

// The picked-up card and the column targets, drawn over the board while a card is held (the Settle motion, see
// practices/design-language.md). Under reduced motion the lift has no scale, and the glide stays.
export const BoardDragLayer = ({ projectPrefix, ticketTypes }: BoardDragLayerProps) => {
  const drag = useContext(BoardDragContext);
  const reduced = useReducedMotion();
  const host = useAnimatedRef<View>();
  const origin = useSharedValue({ x: 0, y: 0 });
  const lift = useSharedValue(0);
  const settle = useSharedValue(0);
  const lifted = useLifted(drag, (held) => held);
  const landing = lifted?.landing;
  // Worklets copy what they close over, so they take the shared values alone, never the whole drag.
  const hovered = drag?.hovered;
  const grab = drag?.grab;
  const tx = drag?.tx;
  const ty = drag?.ty;
  const from = lifted?.from;
  // Over a column the card shrinks around the finger, so the column it is about to land in stays in view.
  const over = useDerivedValue(() =>
    withTiming(hovered && hovered.get() >= 0 ? 1 : 0, { duration: 150, easing: ease.out }),
  );
  const type = ticketTypes?.find((t) => t.id === lifted?.ticket.type_id);

  // The board re-renders around a held card, so the effects read its drag through a ref and run once per lift and landing.
  const latest = useRef(drag);
  useEffect(() => {
    latest.current = drag;
  });

  useEffect(() => {
    if (!lifted || landing !== undefined) return;
    settle.set(0);
    lift.set(withTiming(1, { duration: 150, easing: ease.out }));
    runOnUI(() => {
      const box = measure(host);
      if (box) origin.set({ x: box.pageX, y: box.pageY });
    })();
  }, [lifted, landing, lift, settle, host, origin]);

  useEffect(() => {
    const current = latest.current;
    if (!current || !lifted || landing === undefined) return;
    const { tx, ty, targets, clear, move, statuses } = current;
    const target = targets.get()[landing];
    const status = statuses[landing];
    if (!target || !status) {
      lift.set(withTiming(0, { duration: 150, easing: ease.out }));
      tx.set(withSpring(0, { duration: 300, dampingRatio: 1 }));
      ty.set(withSpring(0, { duration: 300, dampingRatio: 1 }, () => scheduleOnRN(clear)));
      return;
    }
    if (status.id !== lifted.ticket.status) move(lifted.ticket.id, status.id);
    const dx = target.x + target.width / 2 - (lifted.from.x + lifted.from.width / 2);
    const dy = target.y + target.height / 2 - (lifted.from.y + lifted.from.height / 2);
    tx.set(withTiming(dx, { duration: 220, easing: ease.out }));
    ty.set(withTiming(dy, { duration: 220, easing: ease.out }));
    settle.set(withTiming(1, { duration: 220, easing: ease.out }, () => scheduleOnRN(clear)));
  }, [lifted, landing, lift, settle]);

  const cardStyle = useAnimatedStyle(() => {
    if (!from || !grab || !tx || !ty) return {};
    const shrink = 1 - 0.45 * over.get();
    const scale = reduced ? 1 : (1 + 0.03 * lift.get()) * shrink * (1 - 0.6 * settle.get());
    const { x, y } = grab.get();
    return {
      transformOrigin: `${x}px ${y}px`,
      left: from.x - origin.get().x,
      top: from.y - origin.get().y,
      width: from.width,
      opacity: 1 - settle.get(),
      transform: [{ translateX: tx.get() }, { translateY: ty.get() }, { scale }],
    };
  });
  // The elevated shadow is drawn once on its own layer and faded in, so no frame re-renders a shadow.
  const shadowStyle = useAnimatedStyle(() => ({ opacity: lift.get() }));

  return (
    <View ref={host} pointerEvents="box-none" style={StyleSheet.absoluteFill}>
      {drag && lifted && <BoardDropBar />}
      {drag && lifted && (
        <Animated.View pointerEvents="none" style={[{ position: "absolute" }, cardStyle]} className={ticketCardClass}>
          <Animated.View style={[StyleSheet.absoluteFill, shadowStyle, { borderRadius: 9, boxShadow: "0 12px 28px rgba(0, 0, 0, 0.35)" }]} />
          <TicketCardBody
            ticket={lifted.ticket}
            stage={drag.statuses.find((s) => s.id === lifted.ticket.status)?.kind ?? StatusKind.Backlog}
            projectPrefix={projectPrefix}
            typeName={type?.name}
            typeColor={type?.color}
          />
        </Animated.View>
      )}
    </View>
  );
};
