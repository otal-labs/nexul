import { render, screen } from "@testing-library/react-native";
import { Linking } from "react-native";

import { DocBody } from "@/components/docs/DocBody";
import { richBodyToMarkdown } from "@/models/Doc";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
}));

let capturedOnLinkPress: ((url: string) => void) | undefined;
// The shared renderer pulls in native markdown; DocBody only needs its body text and its onLinkPress prop.
jest.mock("@/components/chat/MessageBody", () => {
  const { Text } = jest.requireActual("react-native");
  return {
    MessageBody: ({ body, onLinkPress }: { body: string; onLinkPress?: (url: string) => void }) => {
      capturedOnLinkPress = onLinkPress;
      return <Text>{body}</Text>;
    },
  };
});

const emptyDoc = JSON.stringify({ type: "doc", content: [] });

beforeEach(() => {
  mockPush.mockReset();
  capturedOnLinkPress = undefined;
});

describe("richBodyToMarkdown", () => {
  test("converts headings, lists, code blocks, links and mentions", () => {
    const body = JSON.stringify({
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 1 }, content: [{ type: "text", text: "Title" }] },
        {
          type: "bulletList",
          content: [
            { type: "listItem", content: [{ type: "paragraph", content: [{ type: "text", text: "First item" }] }] },
          ],
        },
        { type: "codeBlock", attrs: { language: "go" }, content: [{ type: "text", text: 'fmt.Println("hi")' }] },
        {
          type: "paragraph",
          content: [
            { type: "text", text: "See " },
            { type: "text", text: "the other doc", marks: [{ type: "link", attrs: { href: "/docs/doc-2" } }] },
            { type: "text", text: " " },
            { type: "mention", attrs: { type: "doc", id: "doc-3", label: "Spec" } },
          ],
        },
      ],
    });

    const markdown = richBodyToMarkdown(body);

    expect(markdown).toContain("# Title");
    expect(markdown).toContain("- First item");
    expect(markdown).toContain('```go\nfmt.Println("hi")\n```');
    expect(markdown).toContain("[the other doc](/docs/doc-2)");
    expect(markdown).toContain("`Spec`");
  });

  test("passes a non-JSON legacy body through unchanged", () => {
    expect(richBodyToMarkdown("Just plain text")).toBe("Just plain text");
  });
});

describe("DocBody", () => {
  test("renders its markdown through the shared renderer", async () => {
    const body = JSON.stringify({
      type: "doc",
      content: [{ type: "paragraph", content: [{ type: "text", text: "Hello" }] }],
    });
    await render(<DocBody body={body} />);

    expect(await screen.findByText("Hello")).toBeTruthy();
  });

  test("a link to another doc opens that doc's screen instead of the system browser", async () => {
    await render(<DocBody body={emptyDoc} />);

    capturedOnLinkPress?.("/docs/doc-2");

    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-2");
  });

  test("a link to a ticket opens the board's ticket screen", async () => {
    await render(<DocBody body={emptyDoc} />);

    capturedOnLinkPress?.("/tickets/t-9");

    expect(mockPush).toHaveBeenCalledWith("/board/ticket/t-9");
  });

  test("an external link opens the system browser", async () => {
    const openURL = jest.spyOn(Linking, "openURL").mockResolvedValue(true);
    await render(<DocBody body={emptyDoc} />);

    capturedOnLinkPress?.("https://example.com");

    expect(openURL).toHaveBeenCalledWith("https://example.com");
  });
});
