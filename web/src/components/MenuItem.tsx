// A row in a popover menu, at the dropdown menu item's metrics so every menu in the app reads alike.
export const menuItemClass =
  "flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 [&_svg:not([class*='text-'])]:text-muted-foreground";

export const menuItemDestructiveClass =
  "text-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:bg-destructive/10 focus-visible:text-destructive [&_svg:not([class*='text-'])]:text-destructive";
