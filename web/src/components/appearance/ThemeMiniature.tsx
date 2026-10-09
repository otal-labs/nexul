import type { ThemePreviewColors } from "@/lib/themePalettes";

interface ThemeMiniatureProps {
  colors: ThemePreviewColors;
  /** Also draws a dialog over a scrim of this strength (0 to 1): the Dialog backdrop preview. */
  scrim?: number;
}

// The app in miniature, in one palette: the canvas, the sidebar with its active marker, a panel with a title, text and a button.
export const ThemeMiniature = ({ colors, scrim }: ThemeMiniatureProps) => {
  const r = Number.parseFloat(colors.radius) || 0;
  return (
    <svg viewBox="0 0 160 100" aria-hidden className="block h-auto w-full" preserveAspectRatio="xMidYMid slice">
      <rect width="160" height="100" fill={colors.background} />
      <rect x="8" y="15" width="2" height="6" rx="1" fill={colors.brand} />
      <rect x="13" y="16.5" width="26" height="3" rx="1.5" fill={colors.foreground} />
      <rect x="13" y="27.5" width="22" height="3" rx="1.5" fill={colors.muted} opacity="0.55" />
      <rect x="13" y="38.5" width="25" height="3" rx="1.5" fill={colors.muted} opacity="0.55" />
      <rect x="13" y="49.5" width="18" height="3" rx="1.5" fill={colors.muted} opacity="0.55" />
      <rect x="48.5" y="8.5" width="104" height="83" rx={r * 1.5} fill={colors.card} stroke={colors.border} />
      <rect x="60" y="21" width="44" height="6" rx={Math.min(r, 3)} fill={colors.foreground} />
      <rect x="60" y="35" width="76" height="3" rx="1.5" fill={colors.muted} opacity="0.6" />
      <rect x="60" y="43" width="60" height="3" rx="1.5" fill={colors.muted} opacity="0.6" />
      <rect x="60" y="68" width="30" height="11" rx={r} fill={colors.brand} />
      <rect x="94.5" y="68.5" width="24" height="10" rx={r} fill="none" stroke={colors.border} />
      {scrim !== undefined && (
        <>
          <rect width="160" height="100" fill="black" opacity={scrim} />
          <rect x="44.5" y="26.5" width="71" height="47" rx={r * 1.5} fill={colors.card} stroke={colors.border} />
          <rect x="53" y="36" width="34" height="5" rx={Math.min(r, 2.5)} fill={colors.foreground} />
          <rect x="53" y="47" width="50" height="3" rx="1.5" fill={colors.muted} opacity="0.6" />
          <rect x="84" y="58" width="24" height="9" rx={r} fill={colors.brand} />
        </>
      )}
    </svg>
  );
};
