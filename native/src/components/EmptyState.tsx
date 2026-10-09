import { Inbox, type LucideIcon } from "lucide-react-native";
import type { ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import Animated, { interpolate, useAnimatedStyle } from "react-native-reanimated";
import Svg, { Circle, Defs, LinearGradient, Stop } from "react-native-svg";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { useEntrance, useReducedMotion } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  title: string;
  message?: string;
  icon?: LucideIcon;
  action?: ReactNode;
  // "compact" inside a section: the same mark at 56, an Inter title, since the display face never goes under 20.
  size?: "default" | "compact";
}

const Ring = ({ id, r, dashed, opacity }: { id: string; r: number; dashed?: boolean; opacity: number }) => {
  const [cool, pink, brand] = useCSSVariable(["--color-field-cool", "--color-field-pink", "--color-brand"]);
  return (
    <Svg style={StyleSheet.absoluteFill} viewBox="0 0 112 112">
      <Defs>
        <LinearGradient id={id} x1="0" y1="1" x2="1" y2="0">
          <Stop offset="0" stopColor={String(cool)} stopOpacity={1} />
          <Stop offset="0.55" stopColor={String(pink)} stopOpacity={1} />
          <Stop offset="1" stopColor={String(brand)} />
        </LinearGradient>
      </Defs>
      <Circle
        cx={56}
        cy={56}
        r={r}
        fill="none"
        stroke={`url(#${id})`}
        strokeOpacity={opacity}
        strokeWidth={1.25}
        {...(dashed && { strokeDasharray: "2 5", strokeLinecap: "round" as const })}
      />
    </Svg>
  );
};

const Dot = ({ cx, cy, r, color }: { cx: number; cy: number; r: number; color: string }) => (
  <Svg style={StyleSheet.absoluteFill} viewBox="0 0 112 112">
    <Circle cx={cx} cy={cy} r={r} fill={color} />
  </Svg>
);

// One layer of the mark: fades in while it moves from its own start (a scale, a turn) to rest.
const useLayer = (duration: number, delay: number, still: boolean, from: { scale?: number; turn?: number }) => {
  const progress = useEntrance(duration, delay, still);
  return useAnimatedStyle(() => {
    const p = progress.get();
    return {
      opacity: p,
      transform: [
        { rotate: `${interpolate(p, [0, 1], [from.turn ?? 0, 0])}deg` },
        { scale: interpolate(p, [0, 1], [from.scale ?? 1, 1]) },
      ],
    };
  });
};

// The icon on two orbits in the light field's colours; on arrival the orbit turns into place, then holds still.
const OrbitMark = ({ icon: Icon, compact, still }: { icon: LucideIcon; compact: boolean; still: boolean }) => {
  const [muted, brand, cool] = useCSSVariable(["--color-muted-foreground", "--color-brand", "--color-field-cool"]);
  const outer = useLayer(520, 0, still, { scale: 0.92 });
  const inner = useLayer(620, 0, still, { turn: -40, scale: 0.9 });
  const ember = useLayer(700, 60, still, { turn: -75 });
  const blue = useLayer(700, 100, still, { turn: -75 });
  const disc = useLayer(320, 0, still, { scale: 0.94 });
  const size = compact ? 56 : 112;
  return (
    <View style={{ width: size, height: size }} className="items-center justify-center">
      <Animated.View style={[StyleSheet.absoluteFill, outer]}>
        <Ring id="outer" r={54} opacity={0.45} />
      </Animated.View>
      <Animated.View style={[StyleSheet.absoluteFill, inner]}>
        <Ring id="inner" r={40} dashed opacity={0.85} />
      </Animated.View>
      <Animated.View style={[StyleSheet.absoluteFill, ember]}>
        <Dot cx={94.2} cy={39.8} r={3} color={String(brand)} />
      </Animated.View>
      <Animated.View style={[StyleSheet.absoluteFill, blue]}>
        <Dot cx={21.8} cy={78} r={2.5} color={String(cool)} />
      </Animated.View>
      <Animated.View
        style={disc}
        className={cn("items-center justify-center rounded-full border border-border bg-card", compact ? "size-7" : "size-12")}
      >
        <Icon size={compact ? 14 : 20} color={String(muted)} />
      </Animated.View>
    </View>
  );
};

const Rise = ({ delay, distance, children, still }: { delay: number; distance: number; children: ReactNode; still: boolean }) => {
  const progress = useEntrance(260, delay, still);
  const style = useAnimatedStyle(() => ({
    opacity: progress.get(),
    transform: [{ translateY: interpolate(progress.get(), [0, 1], [distance, 0]) }],
  }));
  return <Animated.View style={style}>{children}</Animated.View>;
};

// A whole empty screen reads as a first run, not a placeholder box; under reduced motion it fades in once.
export const EmptyState = ({ title, message, icon = Inbox, action, size = "default" }: EmptyStateProps) => {
  const reduced = useReducedMotion();
  const compact = size === "compact";
  const still = reduced || compact;
  // A compact one fades and rises 4px as a block; under reduced motion any size is one 150ms fade.
  const block = useEntrance(compact && !reduced ? 200 : 150, 0, !still);
  const blockStyle = useAnimatedStyle(() => ({
    opacity: block.get(),
    transform: [{ translateY: compact && !reduced ? interpolate(block.get(), [0, 1], [4, 0]) : 0 }],
  }));
  return (
    <Animated.View
      accessible
      role="summary"
      style={blockStyle}
      className={cn("items-center px-8", compact ? "gap-2 py-6" : "flex-1 justify-center pb-16")}
    >
      <OrbitMark icon={icon} compact={compact} still={still} />
      <Rise delay={90} distance={6} still={still}>
        {compact && <Text className="text-center text-sm font-medium">{title}</Text>}
        {!compact && <Text role="heading" className="mt-6 text-center font-display text-[26px] leading-[30px] tracking-[-0.5px]">{title}</Text>}
      </Rise>
      {message && (
        <Rise delay={140} distance={6} still={still}>
          <Text className={cn("max-w-xs text-center text-muted-foreground", compact ? "text-xs" : "mt-2 text-sm leading-5")}>{message}</Text>
        </Rise>
      )}
      {action && (
        <Rise delay={190} distance={6} still={still}>
          <View className={compact ? "mt-1" : "mt-6"}>{action}</View>
        </Rise>
      )}
    </Animated.View>
  );
};
