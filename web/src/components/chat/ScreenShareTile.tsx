import { FocusLayout } from "@livekit/components-react";
import type { TrackReference } from "@livekit/components-core";
import { Maximize2, Minimize2 } from "lucide-react";
import { useRef, useSyncExternalStore } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";

const subscribeFullscreen = (onChange: () => void) => {
  document.addEventListener("fullscreenchange", onChange);
  return () => document.removeEventListener("fullscreenchange", onChange);
};

const isTileFullscreen = () => document.fullscreenElement?.hasAttribute("data-screen-share-tile") ?? false;

interface ScreenShareTileProps {
  trackRef: TrackReference;
}

// The wrapper, not the video, goes full screen so the button stays reachable; the share video is already object-fit: contain.
export const ScreenShareTile = ({ trackRef }: ScreenShareTileProps) => {
  const tile = useRef<HTMLDivElement>(null);
  const fullscreen = useSyncExternalStore(subscribeFullscreen, isTileFullscreen);
  const label = fullscreen ? "Exit full screen" : "Full screen";
  const Icon = fullscreen ? Minimize2 : Maximize2;

  const toggle = () => {
    const request = fullscreen ? document.exitFullscreen() : tile.current?.requestFullscreen();
    request?.catch(() => toast.error("Your browser blocked full screen"));
  };

  return (
    <div
      ref={tile}
      data-screen-share-tile=""
      className="group relative min-h-0 min-w-0 [&:fullscreen]:bg-black [&:fullscreen_video]:bg-black! [&>.lk-participant-tile]:size-full"
      onDoubleClick={toggle}
    >
      <FocusLayout trackRef={trackRef} />
      {document.fullscreenEnabled && (
        <Button
          type="button"
          size="icon"
          variant="outline"
          aria-label={label}
          title={label}
          onClick={toggle}
          onDoubleClick={(e) => e.stopPropagation()}
          className="absolute top-2 right-2 size-8 bg-background/80 opacity-0 focus-visible:opacity-100 group-hover:opacity-100 [@media(hover:none)]:opacity-100"
        >
          <Icon aria-hidden />
        </Button>
      )}
    </div>
  );
};
