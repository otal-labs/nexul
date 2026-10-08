import { useLayoutEffect, type RefObject } from "react";
import { useLocation, useNavigationType } from "react-router";

import { enterPage, lastInputWasKeyboard } from "@/lib/motion";
import { RESERVED_SLUGS } from "@/models/Workspace";

// The page is the path's first segment past the workspace, so a tab, a section or an open record inside it never replays the entrance.
export const usePageEntrance = (frame: RefObject<HTMLElement | null>) => {
  const { pathname } = useLocation();
  const navigationType = useNavigationType();
  const [first = "", second = ""] = pathname.split("/").filter(Boolean);
  const page = RESERVED_SLUGS.has(first) ? first : `${first}/${second}`;

  useLayoutEffect(() => {
    // Back, forward and a reload restore a page the reader has already seen arrive; a keyboard move gets no decoration.
    if (navigationType === "POP" || lastInputWasKeyboard() || !frame.current) return;
    return enterPage(frame.current);
    // Keyed on the page alone: the navigation type only matters at the moment the page changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);
};
