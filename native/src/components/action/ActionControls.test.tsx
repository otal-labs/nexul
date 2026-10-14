import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";
import { View } from "react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { ActionButton } from "@/components/action/ActionButton";
import { ActionRow } from "@/components/action/ActionRow";
import { Text } from "@/components/ui/text";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const REASON = "Your role can't delete roles";

let me: MyWorkspaceInfo;

const renderControl = async (control: React.ReactElement) => {
  jest.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
    return Promise.resolve(me);
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(<QueryClientProvider client={client}>{control}</QueryClientProvider>);
};

const button = (onPress: () => void) => (
  <ActionButton permission="roles:delete" name="Delete role" onPress={onPress}>
    <Text>Delete role</Text>
  </ActionButton>
);

const row = (onPress: () => void) => <ActionRow permission="roles:delete" label="Delete role" onPress={onPress} />;

// The jest host views never report a position, so a measured control answers as one under the top of the window.
beforeAll(() => {
  jest.spyOn(View.prototype, "measureInWindow").mockImplementation((callback) => callback(200, 100, 120, 48));
});

beforeEach(() => {
  me = { role_name: "Developer", permissions: ["roles:delete"], show_locked_actions: true };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe.each([
  { name: "button", draw: button },
  { name: "row", draw: row },
])("$name", ({ draw }) => {
  test("runs its action while live", async () => {
    const onPress = jest.fn();
    await renderControl(draw(onPress));

    await userEvent.press(await screen.findByRole("button", { name: "Delete role" }));

    expect(onPress).toHaveBeenCalledTimes(1);
  });

  test("is absent while hidden", async () => {
    me = { role_name: "Client", permissions: [], show_locked_actions: false };
    await renderControl(draw(jest.fn()));

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await act(async () => {});
    expect(screen.queryByRole("button", { name: "Delete role" })).toBeNull();
  });

  test("does nothing when pressed while locked, and carries its reason for a screen reader", async () => {
    me = { role_name: "Junior Developer", permissions: [], show_locked_actions: true };
    const onPress = jest.fn();
    await renderControl(draw(onPress));

    const control = await screen.findByRole("button", { name: "Delete role" });
    await userEvent.press(control);

    expect(onPress).not.toHaveBeenCalled();
    expect(control).toBeDisabled();
    expect(control.props.accessibilityHint).toBe(REASON);
  });
});

test("a tap on a locked button opens a popover that says why", async () => {
  me = { role_name: "Junior Developer", permissions: [], show_locked_actions: true };
  await renderControl(button(jest.fn()));

  await userEvent.press(await screen.findByRole("button", { name: "Delete role" }));

  expect(await screen.findByText("Delete role is locked")).toBeOnTheScreen();
  expect(screen.getByText(REASON)).toBeOnTheScreen();
});

test("a locked row shows its reason as a second line with no tap", async () => {
  me = { role_name: "Junior Developer", permissions: [], show_locked_actions: true };
  await renderControl(row(jest.fn()));

  expect(await screen.findByText(REASON, { includeHiddenElements: true })).toBeOnTheScreen();
});
