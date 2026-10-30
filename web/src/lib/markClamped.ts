// A ref callback that marks the box data-clamped while its content runs past it, for a "Show all" styled peer-data-clamped.
export const markClamped = (el: HTMLElement | null) => {
  if (!el) return;
  const observer = new ResizeObserver(() => el.toggleAttribute("data-clamped", el.scrollHeight > el.clientHeight + 1));
  observer.observe(el);
  return () => observer.disconnect();
};
