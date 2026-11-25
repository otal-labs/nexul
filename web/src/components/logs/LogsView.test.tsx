import { act, render, screen } from "@testing-library/react";
import { Profiler } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LogsView } from "@/components/logs/LogsView";
import { stackLogSource, type ContainerLogWireLine } from "@/models/ContainerLog";
import { useSessionStore } from "@/stores/sessionStore";
import { useViewAsStore } from "@/stores/viewAsStore";

class FakeSocket {
  static all: FakeSocket[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: { code: number; reason: string }) => void) | null = null;
  constructor(readonly url: string) {
    FakeSocket.all.push(this);
  }
  close() {}
}
const socket = () => FakeSocket.all.at(-1)!;

// Frames run only when a test says so, so a burst of socket messages lands inside one frame.
const frames = new Map<number, FrameRequestCallback>();
let nextFrame = 1;
const runFrame = () =>
  act(async () => {
    const due = [...frames.values()];
    frames.clear();
    due.forEach((run) => run(0));
  });

const wire = (n: number): ContainerLogWireLine => ({ ts: `2026-09-30T10:00:00.${String(n).padStart(9, "0")}Z`, stream: "stdout", line: `line ${n}` });
const emitFrame = (first: number, size: number) =>
  act(async () => socket().onmessage?.({ data: JSON.stringify({ lines: Array.from({ length: size }, (_, i) => wire(first + i)) }) }));

beforeEach(() => {
  FakeSocket.all = [];
  frames.clear();
  vi.stubGlobal("WebSocket", FakeSocket);
  vi.stubGlobal("requestAnimationFrame", (run: FrameRequestCallback) => {
    frames.set(nextFrame, run);
    return nextFrame++;
  });
  vi.stubGlobal("cancelAnimationFrame", (handle: number) => frames.delete(handle));
  useSessionStore.setState({ token: "tok", isLoggedIn: true });
  useViewAsStore.getState().exit();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("LogsView", () => {
  it("commits once for a burst of 100 socket frames within one animation frame", async () => {
    let commits = 0;
    render(
      <Profiler id="logs" onRender={() => (commits += 1)}>
        <LogsView source={stackLogSource("s-1", "web")} />
      </Profiler>,
    );
    await act(async () => socket().onopen?.());
    await emitFrame(0, 5);
    await runFrame();
    await runFrame();

    commits = 0;
    for (let frame = 1; frame <= 100; frame++) await emitFrame(frame * 5, 5);
    expect(commits).toBe(0);
    await runFrame();

    expect(commits).toBe(1);
    expect(screen.getByText("line 504")).toBeInTheDocument();
  });
});
