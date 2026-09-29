import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Room } from "livekit-client";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectedCall } from "@/components/chat/ConnectedCall";
import { Layout } from "@/Layout";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

vi.mock("@/hooks/useLiveEvents", () => ({ useLiveEvents: vi.fn() }));

vi.mock("@/api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/client")>();
  return { ...actual, api: { get: vi.fn(async () => ({ data: [] })), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() } };
});

// jsdom has no WebRTC: the renderer is a marker so the tests count how many are mounted.
vi.mock("@livekit/components-react", () => ({
  RoomAudioRenderer: () => <div data-testid="room-audio" />,
  RoomContext: { Provider: ({ children }: { children: React.ReactNode }) => children },
}));
vi.mock("@/components/chat/VoiceCallStage", () => ({ VoiceCallStage: () => null }));

const room = {} as Room;
const initialState = useVoiceCallStore.getState();

const renderApp = (threadView: boolean) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route
              path="/"
              element={threadView ? <ConnectedCall room={room} resolveLogin={(id) => id} /> : <div>page-content</div>}
            />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("call audio", () => {
  beforeEach(() => {
    useVoiceCallStore.setState(initialState, true);
  });

  it("plays a connected call's audio while the channel's thread view is not open", async () => {
    useVoiceCallStore.setState({ activeConversationId: "c1", status: "connected", room });
    renderApp(false);

    expect(await screen.findByTestId("room-audio")).toBeInTheDocument();
  });

  it("plays it through a single renderer when the thread view is open too", async () => {
    useVoiceCallStore.setState({ activeConversationId: "c1", status: "connected", room });
    renderApp(true);

    await screen.findByTestId("room-audio");
    expect(screen.getAllByTestId("room-audio")).toHaveLength(1);
  });

  it("renders no audio when there is no call", () => {
    renderApp(false);

    expect(screen.queryByTestId("room-audio")).not.toBeInTheDocument();
  });
});
