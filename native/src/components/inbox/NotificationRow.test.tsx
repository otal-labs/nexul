import { render, screen, userEvent } from "@testing-library/react-native";

import { NotificationRow } from "@/components/inbox/NotificationRow";
import type { Notification } from "@/models/Notification";

const unread: Notification = {
  id: "n1",
  user_id: "u1",
  workspace_id: "ws-1",
  kind: "ticket.assigned",
  subject_type: "ticket",
  subject_id: "t-1",
  subject_title: "Write migrations",
  read: false,
  created_at: new Date().toISOString(),
};

describe("NotificationRow", () => {
  test("an unread row shows the unread marker", async () => {
    await render(<NotificationRow notification={unread} onPress={jest.fn()} />);

    expect(screen.getByLabelText("Unread")).toBeTruthy();
    expect(screen.getByText("Write migrations")).toBeTruthy();
  });

  test("a read row shows no unread marker", async () => {
    await render(<NotificationRow notification={{ ...unread, read: true }} onPress={jest.fn()} />);

    expect(screen.queryByLabelText("Unread")).toBeNull();
  });

  test("pressing the row reports the notification", async () => {
    const onPress = jest.fn();
    await render(<NotificationRow notification={unread} onPress={onPress} />);

    await userEvent.setup().press(screen.getByRole("button", { name: /Write migrations/ }));

    expect(onPress).toHaveBeenCalledWith(unread);
  });
});
