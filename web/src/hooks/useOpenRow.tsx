import { useCallback } from "react";
import { useSearchParams } from "react-router";

import { OPEN_PARAM } from "@/models/Agent";

// The one open row (a play or an auto play) lives in the URL, so a link lands on it; opening the open row closes it.
export const useOpenRow = () => {
  const [params, setParams] = useSearchParams();
  const open = params.get(OPEN_PARAM) ?? "";

  const setOpen = useCallback(
    (id: string) =>
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (id === "") next.delete(OPEN_PARAM);
          if (id !== "") next.set(OPEN_PARAM, id);
          return next;
        },
        { replace: true },
      ),
    [setParams],
  );

  const toggle = useCallback((id: string) => setOpen(id === open ? "" : id), [open, setOpen]);
  return { open, setOpen, toggle };
};
