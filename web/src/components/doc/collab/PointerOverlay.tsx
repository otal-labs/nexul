import { useEffect, useState } from "react";
import type { Awareness } from "y-protocols/awareness";

interface RemotePointer {
  clientID: number;
  name: string;
  color: string;
  x: number;
  y: number;
}

interface PointerOverlayProps {
  awareness: Awareness;
  selfID: number;
}

// Reads the awareness "pointer" field directly (no domain event per move) so only this overlay re-renders.
export const PointerOverlay = ({ awareness, selfID }: PointerOverlayProps) => {
  const [pointers, setPointers] = useState<RemotePointer[]>([]);

  useEffect(() => {
    const refresh = () => {
      const out: RemotePointer[] = [];
      for (const [clientID, state] of awareness.getStates()) {
        if (clientID === selfID) continue;
        const s = state as {
          user?: { name?: string; color?: string };
          pointer?: { x?: number; y?: number } | null;
        } | null;
        if (!s?.user?.name || s.pointer == null) continue;
        out.push({
          clientID,
          name: s.user.name,
          color: s.user.color ?? "#6366f1",
          x: s.pointer.x ?? 0,
          y: s.pointer.y ?? 0,
        });
      }
      // Keeps prior array identity when pointers are unchanged so typing doesn't re-render per key.
      setPointers((prev) => (JSON.stringify(prev) === JSON.stringify(out) ? prev : out));
    };
    refresh();
    awareness.on("change", refresh);
    return () => {
      awareness.off("change", refresh);
    };
  }, [awareness, selfID]);

  return (
    <>
      {pointers.map((pointer) => (
        <div
          key={pointer.clientID}
          aria-hidden
          data-testid="remote-pointer"
          className="pointer-events-none absolute top-0 left-0 z-20 transition-transform duration-100 ease-linear motion-reduce:transition-none"
          style={{ transform: `translate(${pointer.x}px, ${pointer.y}px)` }}
        >
          <svg width="14" height="18" viewBox="0 0 14 18" className="drop-shadow-sm">
            <path d="M1 1l12 9-5.2 1L4.6 17z" fill={pointer.color} stroke="#fff" strokeWidth="1" />
          </svg>
          <span
            className="absolute top-4 left-3 rounded-sm px-1.5 py-0.5 text-[11px] leading-tight font-medium whitespace-nowrap text-white"
            style={{ backgroundColor: pointer.color }}
          >
            {pointer.name}
          </span>
        </div>
      ))}
    </>
  );
};
