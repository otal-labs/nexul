import { useEffect, useRef } from "react";
import { Dialog as DialogPrimitive } from "radix-ui";

import { CommandPaletteBody } from "@/components/command/CommandPaletteBody";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";
import type { CommandItem } from "@/models/Command";

// ⌘K or Ctrl+K from anywhere in the signed-in app; the same keys close it again.
export const CommandPalette = () => {
  const open = useCommandPaletteStore((s) => s.open);
  const setOpen = useCommandPaletteStore((s) => s.setOpen);
  const toggle = useCommandPaletteStore((s) => s.toggle);
  const ran = useRef(false);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== "k" || !(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) return;
      event.preventDefault();
      toggle();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [toggle]);

  const select = (item: CommandItem) => {
    ran.current = true;
    setOpen(false);
    item.run();
  };

  return (
    <DialogPrimitive.Root open={open} onOpenChange={setOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="glass-overlay fixed inset-0 z-50 ease-out data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:duration-150 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:duration-200" />
        <DialogPrimitive.Content
          aria-describedby={undefined}
          onOpenAutoFocus={() => (ran.current = false)}
          // After a pick the page or a dialog it opened owns focus; only a dismissal hands it back to the trigger.
          onCloseAutoFocus={(event) => {
            if (ran.current) event.preventDefault();
          }}
          className="glass-popover fixed top-[14dvh] left-1/2 z-50 flex w-[min(40rem,calc(100%-2rem))] -translate-x-1/2 origin-top flex-col overflow-hidden rounded-xl ease-out data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-97 data-[state=closed]:duration-150 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-97 data-[state=open]:duration-200"
        >
          <DialogPrimitive.Title className="sr-only">Command palette</DialogPrimitive.Title>
          <CommandPaletteBody onSelect={select} />
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
};
