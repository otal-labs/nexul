import { createContext, useContext } from "react";
import { useLocation, useParams } from "react-router";

// Set by routes that spell a tab as a static segment because `:tab` would collide with a sibling's `:id`.
export const TabSegmentContext = createContext<string | undefined>(undefined);

// The tab segment ends the path; the default tab has none. `tabPath("versions")` names a tab, `tabPath()` the default.
export const useTabPath = () => {
  const { tab } = useParams();
  const fixed = useContext(TabSegmentContext);
  const { pathname } = useLocation();
  const current = tab ?? fixed;
  const trimmed = pathname.replace(/\/+$/, "");
  const base = current ? trimmed.replace(/\/[^/]+$/, "") : trimmed;
  return { current, tabPath: (value?: string) => (value ? `${base}/${value}` : base) };
};
