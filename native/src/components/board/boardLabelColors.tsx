import { createContext } from "react";

// The board's label colors, read once at the board so no card scans the tickets for them.
export const BoardLabelColorsContext = createContext<Record<string, string> | undefined>(undefined);
