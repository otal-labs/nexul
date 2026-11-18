import { createContext, useContext } from "react";
import { useLocation, useParams } from "react-router";

// Set by routes that spell a tab as a static segment because `:tab` would collide with a sibling's `:id`.
export const TabSegmentContext = createContext<string | undefined>(undefined);

// The base is the path before the tab's segment, so a record opened inside a tab (`access/roles/<id>`) keeps the
// tab row's links right; the default tab has no segment. `tabPath("versions")` names a tab, `tabPath()` the default.
export const useTabPath = () => {
  const { tab } = useParams();
  const fixed = useContext(TabSegmentContext);
  const { pathname } = useLocation();
  const current = tab ?? fixed;
  const segments = pathname.replace(/\/+$/, "").split("/");
  const at = current ? segments.lastIndexOf(current) : -1;
  const base = (at > 0 ? segments.slice(0, at) : segments).join("/");
  return { current, tabPath: (value?: string) => (value ? `${base}/${value}` : base) };
};
