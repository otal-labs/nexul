import { useEffect, useRef } from "react";
import { Dialog as DialogPrimitive } from "radix-ui";

import { CommandPaletteBody } from "@/components/command/CommandPaletteBody";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";

// ⌘K or Ctrl+K from anywhere in the signed-in app; the same keys close it again.
export const CommandPalette = () => {
  const open = useCommandPaletteStore((s) => s.open);
  const setOpen = useCommandPaletteStore((s) => s.setOpen);
  // A key, or a pick that hands over to a page, shows and closes it at once: it is driven a dozen times an hour.
  const instant = useCommandPaletteStore((s) => s.keyed || s.picked);
  const toggle = useCommandPaletteStore((s) => s.toggle);
  // Opened by a shortcut there is no trigger for the dialog to hand focus back to, so it remembers where focus was.
  const returnTo = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== "k" || !(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) return;
      event.preventDefault();
      toggle();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [toggle]);

  return (
    <DialogPrimitive.Root open={open} onOpenChange={setOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay data-instant={instant || undefined} className="data-instant:animate-none! glass-overlay fixed inset-0 z-50 ease-out data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:duration-150 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:duration-200" />
        <DialogPrimitive.Content
          aria-describedby={undefined}
          data-instant={instant || undefined}
          onEscapeKeyDown={(event) => {
            event.preventDefault();
            setOpen(false, true);
          }}
          onOpenAutoFocus={() => {
            returnTo.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
          }}
          // After a pick the page or a dialog it opened owns focus; only a dismissal hands it back.
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            if (!useCommandPaletteStore.getState().picked) returnTo.current?.focus();
          }}
          className="glass-popover fixed top-[14dvh] left-1/2 z-50 flex w-[min(40rem,calc(100%-2rem))] -translate-x-1/2 origin-top flex-col overflow-hidden rounded-xl ease-out data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-97 data-[state=closed]:duration-150 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-97 data-[state=open]:duration-200 data-instant:animate-none!"
        >
          <DialogPrimitive.Title className="sr-only">Command palette</DialogPrimitive.Title>
          <CommandPaletteBody />
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
};
