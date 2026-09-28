import { DarkTheme, DefaultTheme, type Theme } from "expo-router/react-navigation";
import type { ColorValue } from "react-native";
import { useCSSVariable, useUniwind } from "uniwind";

const asColor = (value: string | number | undefined, fallback: ColorValue): ColorValue =>
  typeof value === "string" ? value : fallback;

// Native headers and tab bars cannot read the stylesheet, so they take the same tokens through the theme.
export const useNavigationTheme = (): Theme => {
  const { theme } = useUniwind();
  const base = theme === "dark" ? DarkTheme : DefaultTheme;
  const [background, card, border, text, primary, notification] = useCSSVariable([
    "--color-background",
    "--color-card",
    "--color-border",
    "--color-foreground",
    "--color-primary",
    "--color-destructive",
  ]);

  return {
    ...base,
    colors: {
      background: asColor(background, base.colors.background),
      card: asColor(card, base.colors.card),
      border: asColor(border, base.colors.border),
      text: asColor(text, base.colors.text),
      primary: asColor(primary, base.colors.primary),
      notification: asColor(notification, base.colors.notification),
    },
  };
};
