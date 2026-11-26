const MIN_HEIGHT = 320;
// The page's 32px bottom padding inside the panel.
const BOTTOM_PAD = 32;

// A ref callback: sizes a block down to the panel's bottom, however tall the header above it, never shorter than 20rem;
// outside the page panel (a sheet) the window's bottom stands in for it. Measured again whenever the page above it settles
// or the window resizes.
export const fillToBottom = (el: HTMLDivElement | null) => {
  if (!el) return;
  const frame = el.closest(".app-frame");
  const fit = () => {
    const bottom = frame ? frame.getBoundingClientRect().top + frame.clientTop + frame.clientHeight : window.innerHeight;
    const top = el.getBoundingClientRect().top + (frame?.scrollTop ?? 0);
    const height = `${Math.max(MIN_HEIGHT, Math.floor(bottom - top - BOTTOM_PAD))}px`;
    if (el.style.height !== height) el.style.height = height;
  };
  fit();
  const observer = new ResizeObserver(fit);
  if (frame?.firstElementChild) observer.observe(frame.firstElementChild);
  window.addEventListener("resize", fit);
  return () => {
    observer.disconnect();
    window.removeEventListener("resize", fit);
  };
};
