const Key = ({ children }: { children: string }) => (
  <kbd className="flex h-5 min-w-5 items-center justify-center rounded-sm border border-border bg-muted/60 px-1 font-mono text-[11px] text-muted-foreground">
    {children}
  </kbd>
);

// The keys, so the palette teaches itself; hidden from screen readers, whose own commands already say this.
export const CommandPaletteFooter = () => (
  <div aria-hidden className="flex items-center gap-4 border-t border-border px-4 py-2 text-xs text-muted-foreground">
    <span className="flex items-center gap-1.5">
      <Key>↑</Key>
      <Key>↓</Key>
      to move
    </span>
    <span className="flex items-center gap-1.5">
      <Key>↵</Key>
      to open
    </span>
    <span className="ml-auto flex items-center gap-1.5">
      <Key>esc</Key>
      to close
    </span>
  </div>
);
