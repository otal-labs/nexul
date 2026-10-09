import { useMemo, useState } from "react";
import { FilmGrain, FlowField, MultiPointGradient, Shader, SolidColor } from "shaders/react";

import { useThemeStore } from "@/stores/themeStore";
import { cn } from "@/lib/utils";

interface FieldColors {
  base: string;
  ember: string;
  pink: string;
  blue: string;
}

// The shader takes plain colours, so the tokens are resolved through a probe and their alpha dropped.
const resolveToken = (probe: HTMLElement, token: string) => {
  probe.style.color = `var(${token})`;
  return getComputedStyle(probe).color.replace(/\s*\/\s*[\d.]+%?\s*\)$/, ")");
};

const useFieldColors = (): FieldColors => {
  const theme = useThemeStore((s) => s.theme);
  const themeId = useThemeStore((s) => s.themeId);
  return useMemo(() => {
    const probe = document.createElement("span");
    document.body.append(probe);
    const colors = {
      base: resolveToken(probe, "--background"),
      ember: resolveToken(probe, "--brand"),
      pink: resolveToken(probe, "--field-pink"),
      blue: resolveToken(probe, "--field-cool"),
    };
    probe.remove();
    return colors;
    // theme and themeId are what the tokens are painted from.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [theme, themeId]);
};

const LightField = ({ c, quiet }: { c: FieldColors; quiet: boolean }) => (
  <>
    <SolidColor color={c.base} />
    <MultiPointGradient
      colorA={c.base}
      positionA={{ x: 0.4, y: 0.4 }}
      colorB={c.ember}
      positionB={{ x: 0.94, y: 0.04 }}
      colorC={c.pink}
      positionC={{ x: 0.84, y: 0.4 }}
      colorD={c.blue}
      positionD={{ x: 0.04, y: 0.96 }}
      colorE={c.base}
      positionE={{ x: 0.62, y: 0.82 }}
      smoothness={3}
      colorSpace="oklab"
      opacity={quiet ? 0.22 : 0.46}
    />
    <FlowField strength={0.18} detail={1.1} speed={0.6} evolutionSpeed={0.4} />
    <FilmGrain strength={0.05} bias={1} />
  </>
);

// Loaded on demand, so the shader library only downloads where the live field can run.
export const LiveLightField = ({ quiet }: { quiet: boolean }) => {
  const colors = useFieldColors();
  const [ready, setReady] = useState(false);
  return (
    <Shader
      disableTelemetry
      onReady={() => setReady(true)}
      className={cn("size-full opacity-0 transition-opacity duration-700 ease-out", ready && "opacity-100")}
    >
      <LightField c={colors} quiet={quiet} />
    </Shader>
  );
};
