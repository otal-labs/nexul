import { StyleSheet, View } from "react-native";
import Svg, { Defs, Ellipse, RadialGradient, Stop } from "react-native-svg";
import { useCSSVariable } from "uniwind";

import { svgPaint } from "@/lib/color";

const Glow = ({ id, token }: { id: string; token: string | number | undefined }) => {
  const { color, opacity } = svgPaint(token);
  return (
    <RadialGradient id={id} cx="50%" cy="50%" rx="50%" ry="50%">
      <Stop offset="0" stopColor={color} stopOpacity={opacity} />
      <Stop offset="1" stopColor={color} stopOpacity={0} />
    </RadialGradient>
  );
};

// The web's still light field, drawn once behind a screen: ember top right, a hint of pink, blue bottom left. Never animated.
export const LightField = () => {
  const [warm, pink, cool] = useCSSVariable(["--color-field-warm", "--color-field-pink", "--color-field-cool"]);
  return (
    <View pointerEvents="none" style={StyleSheet.absoluteFill}>
      <Svg width="100%" height="100%" viewBox="0 0 100 100" preserveAspectRatio="none">
        <Defs>
          <Glow id="warm" token={warm} />
          <Glow id="pink" token={pink} />
          <Glow id="cool" token={cool} />
        </Defs>
        <Ellipse cx="88" cy="4" rx="62" ry="30" fill="url(#warm)" />
        <Ellipse cx="58" cy="34" rx="44" ry="22" fill="url(#pink)" />
        <Ellipse cx="6" cy="96" rx="68" ry="34" fill="url(#cool)" />
      </Svg>
    </View>
  );
};
