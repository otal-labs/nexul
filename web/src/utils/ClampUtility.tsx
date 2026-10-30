// Marks the box data-clamped while its text runs past it; watching the content too catches text that grows while clamped.
// Fractional heights, because the rounded scrollHeight runs 2px past clientHeight on one line at some page zooms.
export const markClamped = (el: HTMLDivElement | null) => {
  const content = el?.firstElementChild;
  if (!el || !content) return;
  const mark = () =>
    el.toggleAttribute("data-clamped", content.getBoundingClientRect().height > el.getBoundingClientRect().height + 1);
  const observer = new ResizeObserver(mark);
  observer.observe(el);
  observer.observe(content);
  return () => observer.disconnect();
};
