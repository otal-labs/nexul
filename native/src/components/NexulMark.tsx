import Svg, { Circle, Path } from "react-native-svg";

interface NexulMarkProps {
  size: number;
  color: string;
}

// assets/mark.svg drawn in a token colour.
export const NexulMark = ({ size, color }: NexulMarkProps) => (
  <Svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke={color} strokeWidth={2.25} strokeLinecap="round" strokeLinejoin="round">
    <Circle cx={12} cy={12} r={8.5} />
    <Path d="M7.6 7.6L12 12l4.4 4.4" />
    <Circle cx={12} cy={12} r={2.8} fill={color} stroke="none" />
    <Circle cx={6} cy={6} r={2.2} />
    <Circle cx={18} cy={18} r={2.2} />
  </Svg>
);
