import { useRef } from "react";

// Radix returns focus to a menu's trigger on close, which Chrome paints as keyboard focus even after a click; this
// hands focus back without the ring when the menu was used with a pointer, and leaves keyboard use alone.
export const usePointerCloseFocus = <T extends HTMLElement>() => {
  const trigger = useRef<T>(null);
  const pointer = useRef(false);
  const byPointer = () => {
    pointer.current = true;
  };
  const byKeyboard = () => {
    pointer.current = false;
  };
  return {
    triggerProps: { ref: trigger, onPointerDown: byPointer, onKeyDown: byKeyboard },
    contentProps: {
      onPointerDown: byPointer,
      onKeyDown: byKeyboard,
      onCloseAutoFocus: (event: Event) => {
        if (!pointer.current) return;
        event.preventDefault();
        trigger.current?.focus({ preventScroll: true, focusVisible: false });
      },
    },
  };
};
