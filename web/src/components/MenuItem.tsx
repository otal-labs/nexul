// A row in a popover menu, at the dropdown menu item's metrics so every menu in the app reads alike: 32px, a muted
// 16px icon column, the label, then a trailing hint or check. The 5px corner sits concentric in the panel's 9px.
export const menuItemClass =
  "flex min-h-8 items-center gap-2.5 rounded-[5px] px-2 py-1.5 text-left text-sm text-foreground outline-none transition-colors duration-[120ms] ease-standard hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 [&_svg:not([class*='text-'])]:text-muted-foreground";

export const menuItemDestructiveClass =
  "text-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:bg-destructive/10 focus-visible:text-destructive [&_svg:not([class*='text-'])]:text-destructive";

// Between groups of rows, edge to edge across the panel's 4px padding.
export const menuSeparatorClass = "-mx-1 my-1 h-px shrink-0 bg-border";

export const MenuSeparator = () => <div role="separator" className={menuSeparatorClass} />;
