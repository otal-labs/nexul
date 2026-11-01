import { prefersReducedMotion } from "@/lib/motion";

// Swing (the Board drop lock in practices/design-language.md): the lifted card hangs from where it was picked up and leans with the pointer's speed.
const MAX_DEG = 5;
const DEG_PER_PX_MS = 3.5;
const STIFFNESS = 200;
const DAMPING = 2 * 0.45 * Math.sqrt(STIFFNESS);
const SMOOTHING_MS = 40;
// Level inside the 220ms drop glide, so the card waiting in the slot takes over flat.
const LEVEL_MS = 200;

export interface Tilt {
  angle: number;
  spin: number;
  velocity: number;
  x: number;
  landFrom: number | null;
}

export const restingTilt = (x: number): Tilt => ({ angle: 0, spin: 0, velocity: 0, x, landFrom: null });

// One frame: the ms since the last, the pointer's x now, and the ms since the card was let go (null while held).
export const stepTilt = (tilt: Tilt, dt: number, x: number, sinceRelease: number | null): Tilt => {
  if (dt <= 0) return tilt;
  const velocity = tilt.velocity + ((x - tilt.x) / dt - tilt.velocity) * (1 - Math.exp(-dt / SMOOTHING_MS));
  if (sinceRelease !== null) {
    const landFrom = tilt.landFrom ?? tilt.angle;
    const progress = Math.min(sinceRelease / LEVEL_MS, 1);
    return { angle: landFrom * (1 - progress) ** 3, spin: 0, velocity, x, landFrom };
  }
  // tanh eases a fast fling into the cap instead of stopping it dead there.
  const target = MAX_DEG * Math.tanh((DEG_PER_PX_MS * velocity) / MAX_DEG);
  const s = dt / 1000;
  const spin = tilt.spin + (-STIFFNESS * (tilt.angle - target) - DAMPING * tilt.spin) * s;
  return { angle: tilt.angle + spin * s, spin, velocity, x, landFrom: null };
};

interface Box {
  left: number;
  top: number;
  width: number;
  height: number;
}

const grabPoint = (event: Event | null) => {
  if (typeof TouchEvent !== "undefined" && event instanceof TouchEvent) {
    const touch = event.touches[0];
    return touch && { x: touch.clientX, y: touch.clientY };
  }
  if (event instanceof MouseEvent) return { x: event.clientX, y: event.clientY };
  return undefined;
};

// A keyboard drag moves in jumps, so it gets no tilt; reduced motion keeps the card level.
export const followTilt = (card: HTMLElement | null, activator: Event | null, source: Box | null) => {
  const grab = grabPoint(activator);
  if (!card || !grab || !source || prefersReducedMotion()) return undefined;
  const dx = grab.x - (source.left + source.width / 2);
  const dy = grab.y - (source.top + source.height / 2);

  // The pointer, not dnd-kit's delta: the delta carries the board's autoscroll, which the fixed overlay never moves with.
  let pointerX = grab.x;
  let releasedAt: number | null = null;
  const onMouse = (e: MouseEvent) => (pointerX = e.clientX);
  const onTouch = (e: TouchEvent) => (pointerX = e.touches[0]?.clientX ?? pointerX);
  // The drop animation renders the overlay under a nulled dnd context, so the release is read here, not from dnd-kit.
  const onRelease = () => (releasedAt ??= performance.now());
  const onKey = (e: KeyboardEvent) => e.key === "Escape" && onRelease();
  const listening = new AbortController();
  const options = { capture: true, passive: true, signal: listening.signal };
  window.addEventListener("mousemove", onMouse, options);
  window.addEventListener("touchmove", onTouch, options);
  window.addEventListener("mouseup", onRelease, options);
  window.addEventListener("touchend", onRelease, options);
  window.addEventListener("touchcancel", onRelease, options);
  window.addEventListener("keydown", onKey, options);

  let tilt = restingTilt(pointerX);
  let last = performance.now();
  let frame = 0;
  const tick = () => {
    frame = requestAnimationFrame(tick);
    const now = performance.now();
    // Capped: a stiffer spring integrated over one long frame flies off to thousands of degrees.
    tilt = stepTilt(tilt, Math.min(now - last, 50), pointerX, releasedAt === null ? null : now - releasedAt);
    last = now;
    // Rotating about the centre and shifting by what that moved the grab point pivots the card there, while the lift's scale stays centred.
    const rad = (tilt.angle * Math.PI) / 180;
    const tx = dx - (dx * Math.cos(rad) - dy * Math.sin(rad));
    const ty = dy - (dx * Math.sin(rad) + dy * Math.cos(rad));
    card.style.rotate = `${tilt.angle.toFixed(3)}deg`;
    card.style.translate = `${tx.toFixed(2)}px ${ty.toFixed(2)}px`;
  };
  frame = requestAnimationFrame(tick);
  return () => {
    cancelAnimationFrame(frame);
    listening.abort();
  };
};
