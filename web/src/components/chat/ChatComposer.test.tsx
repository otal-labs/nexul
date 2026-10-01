import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatComposer } from "@/components/chat/ChatComposer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast }));

const people = {
  people: [
    { user_id: "u1", login: "onik97", display_name: "", avatar_url: "" },
    { user_id: "u2", login: "olive", display_name: "", avatar_url: "" },
  ],
};

const renderComposer = (onSend = vi.fn(async () => {})) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const result = render(
    <ChatComposer workspaceId="ws-1" conversationId="c-1" onSend={onSend} />,
    { wrapper },
  );
  return { ...result, onSend };
};

const pngFile = (name = "shot.png") => new File(["png"], name, { type: "image/png" });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: people });
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.delete).mockResolvedValue({ data: undefined });
  toast.success.mockReset();
  toast.error.mockReset();
  globalThis.URL.createObjectURL = vi.fn(() => "blob:preview");
  globalThis.URL.revokeObjectURL = vi.fn();
});

describe("ChatComposer", () => {
  it("sends the trimmed body on Enter and clears the input", async () => {
    const user = userEvent.setup();
    const { onSend } = renderComposer();
    const input = screen.getByLabelText("Message");
    await user.type(input, "  hello team  ");
    await user.keyboard("{Enter}");
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("hello team"));
    expect(input).toHaveValue("");
  });

  it("does not send on Shift+Enter, and inserts a newline instead", async () => {
    const user = userEvent.setup();
    const { onSend } = renderComposer();
    const input = screen.getByLabelText("Message");
    await user.type(input, "line one{Shift>}{Enter}{/Shift}line two");
    expect(onSend).not.toHaveBeenCalled();
    expect(input).toHaveValue("line one\nline two");
  });

  it("does not send an empty or whitespace-only message", async () => {
    const user = userEvent.setup();
    const { onSend } = renderComposer();
    await user.type(screen.getByLabelText("Message"), "{Enter}");
    expect(onSend).not.toHaveBeenCalled();
    await user.type(screen.getByLabelText("Message"), "   ");
    await user.keyboard("{Enter}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("opens the @ mention picker with Agent plus workspace members, prefix-filtered while typing", async () => {
    const user = userEvent.setup();
    renderComposer();
    const input = screen.getByLabelText("Message");
    await user.type(input, "@o");
    expect(await screen.findByRole("option", { name: /@onik97/ })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /@olive/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /@Agent/ })).not.toBeInTheDocument();

    await user.type(input, "ni");
    expect(screen.getByRole("option", { name: /@onik97/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /@olive/ })).not.toBeInTheDocument();
  });

  it("picks a mention with Enter, inserting the handle and closing the picker", async () => {
    const user = userEvent.setup();
    renderComposer();
    const input = screen.getByLabelText("Message");
    await user.type(input, "hey @oni");
    await screen.findByRole("option", { name: /@onik97/ });
    await user.keyboard("{Enter}");
    expect(input).toHaveValue("hey @onik97 ");
    expect(screen.queryByRole("listbox", { name: "Mention suggestions" })).not.toBeInTheDocument();
  });

  it("uploads a picked image against the conversation and sends a body with the markdown line", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "a-1", conversation_id: "c-1", name: "shot.png", content_type: "image/png", size: 3 },
    });
    const user = userEvent.setup();
    const { onSend } = renderComposer();

    await user.upload(screen.getByLabelText("Choose image files"), pngFile());

    await waitFor(() => expect(api.post).toHaveBeenCalled());
    const form = vi.mocked(api.post).mock.calls[0]?.[1] as FormData;
    expect(vi.mocked(api.post).mock.calls[0]?.[0]).toBe("/api/attachments");
    expect(form.get("conversation_id")).toBe("c-1");

    await screen.findByAltText("shot.png");
    await user.type(screen.getByLabelText("Message"), "{Enter}");
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("![shot.png](/api/attachments/a-1)"));
  });

  it("toasts and uploads nothing when a non-image file is pasted", async () => {
    renderComposer();
    const file = new File(["pdf"], "spec.pdf", { type: "application/pdf" });

    fireEvent.paste(screen.getByLabelText("Message"), {
      clipboardData: { files: [file], items: [], types: ["Files"], getData: () => "" },
    });

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Only images can be attached"));
    expect(api.post).not.toHaveBeenCalled();
  });

  it("removes a pending attachment by deleting it and dropping its thumbnail", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "a-1", conversation_id: "c-1", name: "shot.png", content_type: "image/png", size: 3 },
    });
    const user = userEvent.setup();
    renderComposer();

    await user.upload(screen.getByLabelText("Choose image files"), pngFile());
    await screen.findByAltText("shot.png");

    await user.click(screen.getByRole("button", { name: "Remove shot.png" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/attachments/a-1"));
    expect(screen.queryByAltText("shot.png")).not.toBeInTheDocument();
  });

  it("does not send on Enter while an upload is in flight, then sends once it finishes", async () => {
    let resolveUpload: (value: { data: unknown }) => void = () => {};
    vi.mocked(api.post).mockReturnValue(
      new Promise((resolve) => {
        resolveUpload = resolve;
      }) as ReturnType<typeof api.post>,
    );
    const user = userEvent.setup();
    const { onSend } = renderComposer();
    const input = screen.getByLabelText("Message");

    await user.type(input, "look");
    await user.upload(screen.getByLabelText("Choose image files"), pngFile());
    await user.type(input, "{Enter}");
    expect(onSend).not.toHaveBeenCalled();

    resolveUpload({ data: { id: "a-1", conversation_id: "c-1", name: "shot.png", content_type: "image/png", size: 3 } });
    await screen.findByAltText("shot.png");
    await user.type(input, "{Enter}");
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("look\n![shot.png](/api/attachments/a-1)"));
  });

  it("sends an image-only message with no text", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "a-1", conversation_id: "c-1", name: "shot.png", content_type: "image/png", size: 3 },
    });
    const user = userEvent.setup();
    const { onSend } = renderComposer();

    await user.upload(screen.getByLabelText("Choose image files"), pngFile());
    await screen.findByAltText("shot.png");
    await user.type(screen.getByLabelText("Message"), "{Enter}");
    await waitFor(() => expect(onSend).toHaveBeenCalledWith("![shot.png](/api/attachments/a-1)"));
  });
});
