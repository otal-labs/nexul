// The name column: sticky on the left, opaque so cells scrolled under it never show, casting its edge once scrolled.
export const stickyCell = "sticky left-0 z-10 w-50 min-w-50 max-w-50 bg-card lg:w-60 lg:min-w-60 lg:max-w-60 grid-edge";
// Every column head, the roles' names and the name and last columns' alike, so they share one style and one baseline.
export const headText = "text-xs leading-tight font-medium text-muted-foreground";
export const leadStrip = "border-b border-border bg-muted/40 dark:bg-muted/80";
// A tint laid over the sticky cell's opaque card fill, so ticks scrolled under it never show through.
export const stickyStrip = "bg-linear-to-r from-muted/40 to-muted/40 dark:from-muted/80 dark:to-muted/80";
// The lead row's cell in the column pinned right: opaque like the name column, so the counts it covers never show through.
export const pinnedStrip = `grid-edge-end sticky right-0 z-10 border-b border-border bg-card ${stickyStrip}`;
export const stickyHover = "group-hover/row:bg-linear-to-r group-hover/row:from-accent/40 group-hover/row:to-accent/40";
// The name's text sits out of flow, so a long name truncates inside the column instead of widening it.
export const nameText = "absolute inset-x-0 top-1/2 -translate-y-1/2 truncate";
// The empty last column of a grid with nothing pinned right: 16px that fade the columns passing under it while more lie past the edge.
export const fadeEndCell = "grid-fade-end sticky right-0 z-10 w-4 min-w-4 border-b border-border p-0";
// A grid with nothing pinned right takes its spare width here, after the last role.
export const fillerCell = "border-b border-border";
// The grid scrolls sideways a whole role column at a time, so it never rests on half a column under the sticky ones.
export const snapScroller = "snap-x snap-proximity scroll-ps-50 lg:scroll-ps-60";
// The role matrix has no filler, so its columns share the spare width; its end fade is wide enough to read as more to scroll.
export const matrixFadeWidth = "w-12 min-w-12";
// Below 1024px the matrix's area column narrows, its names wrapping to two lines, so another role column fits.
export const matrixAreaWidth = "w-44 min-w-44 max-w-44";
