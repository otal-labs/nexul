import { lazy } from "react";

// livekit-client + @livekit/components-react are ~650 kB raw; load only once a voice channel opens.
export const LazyVoiceCallSection = lazy(() =>
  import("@/components/chat/VoiceCallSection").then((m) => ({ default: m.VoiceCallSection })),
);
