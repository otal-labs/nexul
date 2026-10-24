import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";
import { Linking } from "react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { MeetingScreen } from "@/components/calendar/MeetingScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), put: jest.fn() } }));

let mockParams: Record<string, string> = { id: "m-1" };
jest.mock("expo-router", () => ({ useLocalSearchParams: () => mockParams }));

notifyManager.setScheduler((callback) => callback());

const people = [
  { user_id: "me", login: "onik97", display_name: "Onik", avatar_url: "" },
  { user_id: "ana", login: "ana", display_name: "Ana", avatar_url: "" },
  { user_id: "bo", login: "bo", display_name: "Bo", avatar_url: "" },
  { user_id: "cy", login: "cy", display_name: "Cy", avatar_url: "" },
];

const meeting = (extra: object = {}) => ({
  id: "m-1", workspace_id: "ws-1", organiser_id: "ana", title: "Sprint planning", notes: "Bring the estimates.",
  starts_at: "2026-10-28T09:00:00Z", ends_at: "2026-10-28T10:00:00Z", timezone: "UTC", voice_channel_id: "", link: "",
  repeats: "weekly", remind_minutes: 10, created_at: "", updated_at: "",
  invitees: [{ user_id: "me", reply: "pending" }, { user_id: "bo", reply: "accepted" }, { user_id: "cy", reply: "declined" }],
  ...extra,
});

let current: ReturnType<typeof meeting>;

const serve = (path: string) => {
  if (path === "/api/auth/me") return Promise.resolve({ user: { id: "me", login: "onik97", name: "Onik" } });
  if (path === "/api/meetings/m-1") return Promise.resolve(current);
  if (path === "/api/workspaces/ws-1/people") return Promise.resolve({ people });
  if (path === "/api/chat/conversations?workspace_id=ws-1") return Promise.resolve([{ id: "vc-1", workspace_id: "ws-1", kind: "voice_channel", name: "Standup room" }]);
  return Promise.reject(new Error(`unexpected GET ${path}`));
};

const renderScreen = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MeetingScreen />
    </QueryClientProvider>,
  );

beforeEach(() => {
  mockParams = { id: "m-1" };
  current = meeting();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(serve);
  jest.mocked(api.put).mockReset();
});

describe("MeetingScreen", () => {
  test("a meeting the viewer can't read shows its not-found message", async () => {
    jest.mocked(api.get).mockImplementation((path: string) => (path === "/api/meetings/m-1" ? Promise.reject(new ApiError(404, null, "gone")) : serve(path)));
    await renderScreen();

    expect(await screen.findByText("This meeting doesn't exist, or you're not in it.")).toBeTruthy();
  });

  test("shows the title, repeat, organiser, notes and the invitees grouped by their reply", async () => {
    await renderScreen();

    expect(await screen.findByText("Sprint planning")).toBeTruthy();
    expect(screen.getByText("Weekly")).toBeTruthy();
    expect(screen.getByText("Bring the estimates.")).toBeTruthy();
    expect(await screen.findByText("Ana")).toBeTruthy();
    expect(screen.getByText("Accepted · 1")).toBeTruthy();
    expect(screen.getByText("Declined · 1")).toBeTruthy();
    expect(screen.getByText("No reply · 1")).toBeTruthy();
    expect(screen.getByText("Bo")).toBeTruthy();
    expect(screen.getByText("Cy")).toBeTruthy();
  });

  test("a voice meeting names its channel", async () => {
    current = meeting({ voice_channel_id: "vc-1" });
    await renderScreen();

    expect(await screen.findByText("Standup room")).toBeTruthy();
  });

  test("a link meeting opens the link in the browser", async () => {
    current = meeting({ link: "https://meet.example.com/abc" });
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue(true);
    await renderScreen();

    await userEvent.setup().press(await screen.findByRole("link", { name: "Open meet.example.com" }));

    expect(open).toHaveBeenCalledWith("https://meet.example.com/abc");
  });

  test("Accept sends the reply and the screen then shows it as the current one", async () => {
    jest.mocked(api.put).mockResolvedValue(meeting({ invitees: [{ user_id: "me", reply: "accepted" }, { user_id: "bo", reply: "accepted" }, { user_id: "cy", reply: "declined" }] }));
    await renderScreen();
    expect(await screen.findByText("You haven't replied")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: "Accept" }));

    expect(api.put).toHaveBeenCalledWith("/api/meetings/m-1/reply", { reply: "accepted" });
    expect(await screen.findByText("You accepted")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Accept" }).props.accessibilityState).toMatchObject({ selected: true });
  });

  test("Decline sends the decline, and a refused reply shows its error and keeps the old answer", async () => {
    jest.mocked(api.put).mockRejectedValue(new ApiError(500, { message: "Try again later", code: "x" }, "failed"));
    await renderScreen();
    await screen.findByText("You haven't replied");

    await userEvent.setup().press(screen.getByRole("button", { name: "Decline" }));

    expect(api.put).toHaveBeenCalledWith("/api/meetings/m-1/reply", { reply: "declined" });
    expect(await screen.findByText("Try again later")).toBeTruthy();
    expect(screen.getByText("You haven't replied")).toBeTruthy();
  });

  test("the answer already given is not sent again", async () => {
    current = meeting({ invitees: [{ user_id: "me", reply: "declined" }] });
    await renderScreen();
    await screen.findByText("You declined");

    await userEvent.setup().press(screen.getByRole("button", { name: "Decline" }));

    expect(api.put).not.toHaveBeenCalled();
  });

  test("the organiser, who isn't invited, gets no reply buttons", async () => {
    current = meeting({ invitees: [{ user_id: "bo", reply: "accepted" }] });
    await renderScreen();
    await screen.findByText("Sprint planning");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/me"));

    expect(screen.queryByRole("button", { name: "Accept" })).toBeNull();
  });

  test("a cancelled meeting says so and takes no reply", async () => {
    current = meeting({ cancelled_at: "2026-10-27T08:00:00Z" });
    await renderScreen();

    expect(await screen.findByText("Cancelled")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Accept" })).toBeNull();
  });
});
