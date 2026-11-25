import type { TrailState } from "@nexul/client-core/trail";

// Mirrors pairing.SharedRun: a play run someone else ran on the viewer's computer, never what it did.
export interface SharedRun {
  trail_id: string;
  play: string;
  starter_id: string;
  starter_name: string;
  state: TrailState;
  started_at: string;
  ended_at: string | null;
}
