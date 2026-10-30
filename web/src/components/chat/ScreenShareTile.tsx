import { VideoTrack } from "@livekit/components-react";
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

// The wrapper, not the video, goes full screen so the button stays reachable; the screen keeps its whole frame on the media surface.
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
      className="relative min-h-0 min-w-0 bg-media"
      onDoubleClick={toggle}
    >
      <VideoTrack trackRef={trackRef} className="size-full object-contain" />
      {document.fullscreenEnabled && (
        <Button
          type="button"
          size="icon"
          variant="ghost"
          aria-label={label}
          title={label}
          onClick={toggle}
          onDoubleClick={(e) => e.stopPropagation()}
          className="dark absolute top-2 right-2 size-8 bg-media/60 text-white hover:bg-media/80 hover:text-white"
        >
          <Icon aria-hidden />
        </Button>
      )}
    </div>
  );
};
