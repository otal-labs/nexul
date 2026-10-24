import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { AgendaScreen } from "@/components/calendar/AgendaScreen";
import { agendaWindow } from "@/lib/agenda";
import { formatMessageTime } from "@/lib/time";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

notifyManager.setScheduler((callback) => callback());

const day = (offset: number, hour: number) => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate() + offset, hour, 0).toISOString();
};
const dayKeyAt = (offset: number) => day(offset, 12).slice(0, 10);

const meeting = (id: string, title: string, extra: object = {}) => ({
  id, workspace_id: "ws-1", organiser_id: "u1", title, notes: "", starts_at: day(0, 9), ends_at: day(0, 10), timezone: "UTC",
  voice_channel_id: "", link: "", repeats: "none", remind_minutes: 10, invitees: [], created_at: "", updated_at: "", ...extra,
});
const occurrence = (id: string, offset: number, hour: number) => ({
  meeting_id: id, slot: day(offset, hour), starts_at: day(offset, hour), ends_at: day(offset, hour + 1),
});

let range: object;
let dueItems: object[];
let dueFailure: Error | undefined;
const window = agendaWindow(new Date());

const serve = (path: string) => {
  if (path === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
  if (path === `/api/workspaces/ws-1/meetings?from=${encodeURIComponent(window.from)}&to=${encodeURIComponent(window.to)}`) return Promise.resolve(range);
  if (path === `/api/tickets/due?workspace_id=ws-1&from=${window.fromDay}&to=${window.toDay}`) {
    return dueFailure ? Promise.reject(dueFailure) : Promise.resolve({ items: dueItems });
  }
  if (path === "/api/projects/p-1") return Promise.resolve({ id: "p-1", name: "Nexul", prefix: "NEX" });
  if (path === "/api/chat/conversations?workspace_id=ws-1") return Promise.resolve([{ id: "vc-1", workspace_id: "ws-1", kind: "voice_channel", name: "Standup room" }]);
  return Promise.reject(new Error(`unexpected GET ${path}`));
};

const renderScreen = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AgendaScreen />
    </QueryClientProvider>,
  );

beforeEach(() => {
  range = { meetings: [], occurrences: [] };
  dueItems = [];
  dueFailure = undefined;
  mockPush.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(serve);
});

describe("AgendaScreen", () => {
  test("a failed read shows its error, not an empty agenda", async () => {
    dueFailure = new Error("due route down");
    await renderScreen();

    expect(await screen.findByText("due route down")).toBeTruthy();
    expect(screen.queryByText("Nothing coming up")).toBeNull();
  });

  test("shows the empty state when nothing is coming up", async () => {
    await renderScreen();

    expect(await screen.findByText("Nothing coming up")).toBeTruthy();
  });

  test("groups meetings and due tickets under the day they fall on, each meeting with its time and where", async () => {
    range = {
      meetings: [
        meeting("m-1", "Standup", { voice_channel_id: "vc-1" }),
        meeting("m-2", "Planning", { link: "https://meet.example.com/abc" }),
        meeting("m-3", "Retro", { cancelled_at: day(-1, 9) }),
      ],
      occurrences: [occurrence("m-1", 0, 9), occurrence("m-2", 1, 14), occurrence("m-3", 1, 16)],
    };
    dueItems = [{ id: "t-9", project_id: "p-1", title: "Ship the report", number: 9, due_on: dayKeyAt(2) }];
    await renderScreen();

    await screen.findByText("Standup");
    expect(screen.getByText("Today")).toBeTruthy();
    expect(screen.getByText("Tomorrow")).toBeTruthy();
    expect(screen.getByText(`${formatMessageTime(day(0, 9))}–${formatMessageTime(day(0, 10))}`)).toBeTruthy();
    expect(await screen.findByText("Standup room")).toBeTruthy();
    expect(screen.getByText("meet.example.com")).toBeTruthy();
    expect(screen.getByText("Ship the report")).toBeTruthy();
    expect(await screen.findByText("NEX-9")).toBeTruthy();
  });

  test("a cancelled meeting stays on the agenda marked cancelled", async () => {
    range = { meetings: [meeting("m-3", "Retro", { cancelled_at: day(-1, 9) })], occurrences: [occurrence("m-3", 0, 16)] };
    await renderScreen();

    expect(await screen.findByText("Retro")).toBeTruthy();
    expect(screen.getByText("Cancelled")).toBeTruthy();
  });

  test("a meeting opens its screen on the occurrence tapped, and a due ticket opens the ticket", async () => {
    range = { meetings: [meeting("m-1", "Standup")], occurrences: [occurrence("m-1", 1, 9)] };
    dueItems = [{ id: "t-9", project_id: "p-1", title: "Ship the report", number: 9, due_on: dayKeyAt(1) }];
    await renderScreen();
    const user = userEvent.setup();

    await user.press(await screen.findByRole("button", { name: /Standup/ }));
    expect(mockPush).toHaveBeenCalledWith(`/more/calendar/m-1?at=${encodeURIComponent(day(1, 9))}`);

    await user.press(screen.getByRole("button", { name: /Ship the report/ }));
    expect(mockPush).toHaveBeenCalledWith("/board/ticket/t-9", { withAnchor: true });
  });

  test("a voice meeting is the only one that reads the channel list", async () => {
    range = { meetings: [meeting("m-2", "Planning", { link: "https://meet.example.com/abc" })], occurrences: [occurrence("m-2", 0, 14)] };
    await renderScreen();
    await screen.findByText("Planning");

    expect(jest.mocked(api.get).mock.calls.map(([path]) => path).filter((path) => path.startsWith("/api/chat"))).toEqual([]);
  });
});
