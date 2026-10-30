import { ConnectionQuality } from "livekit-client";

export const isWeakConnection = (quality: ConnectionQuality) => quality === ConnectionQuality.Poor || quality === ConnectionQuality.Lost;
