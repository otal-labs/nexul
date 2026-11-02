import Bot from "lucide-react-native/icons/bot";
import Svg, { Path } from "react-native-svg";
import { useCSSVariable, useUniwind } from "uniwind";

import { providerMark, type MarkFill } from "@nexul/client-core/providerMarks";

interface HarnessProviderMarkProps {
  driver: string;
  size?: number;
}

// The web's provider marks in the brand's own fills for the theme; one without a mark gets a generic bot glyph.
export const HarnessProviderMark = ({ driver, size = 14 }: HarnessProviderMarkProps) => {
  const { theme } = useUniwind();
  const [ink, muted] = useCSSVariable(["--color-foreground", "--color-muted-foreground"]);
  const mark = providerMark(driver);
  if (!mark) return <Bot color={String(muted)} size={size} />;
  const paint = (fill: MarkFill | null): string => {
    if (fill === null) return String(ink);
    return theme === "dark" ? fill.dark : fill.light;
  };
  const rule = mark.evenOdd ? "evenodd" : "nonzero";
  return (
    <Svg width={size} height={size} viewBox={mark.viewBox} aria-hidden>
      {mark.hole && <Path d={mark.hole.d} fill={paint(mark.hole.fill)} />}
      {mark.paths.map((d) => (
        <Path key={d} d={d} fill={paint(mark.fill)} fillRule={rule} clipRule={rule} />
      ))}
    </Svg>
  );
};
