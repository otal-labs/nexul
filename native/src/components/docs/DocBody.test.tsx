import { render, screen, userEvent } from "@testing-library/react-native";

import { DocBody } from "@/components/docs/DocBody";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ push: mockPush }),
}));

jest.mock("@/stores/sessionStore", () => ({
  useSessionStore: (selector: (state: { host: string }) => unknown) =>
    selector({ host: "https://nexul.example.com" }),
  readSessionToken: () => "tok_abc",
}));

const richBody = JSON.stringify({
  type: "doc",
  content: [
    { type: "heading", attrs: { level: 1 }, content: [{ type: "text", text: "Title" }] },
    {
      type: "paragraph",
      content: [
        { type: "text", text: "See " },
        { type: "text", text: "the other doc", marks: [{ type: "link", attrs: { href: "/docs/doc-2" } }] },
        { type: "text", text: " and " },
        { type: "mention", attrs: { type: "doc", id: "doc-3", label: "Spec" } },
      ],
    },
    {
      type: "bulletList",
      content: [{ type: "listItem", content: [{ type: "paragraph", content: [{ type: "text", text: "First item" }] }] }],
    },
    { type: "codeBlock", attrs: { language: "go" }, content: [{ type: "text", text: 'fmt.Println("hi")' }] },
    { type: "image", attrs: { src: "/api/attachments/att-1", alt: "Diagram" } },
  ],
});

beforeEach(() => {
  mockPush.mockReset();
});

describe("DocBody", () => {
  test("renders headings, lists, code blocks, mentions and images", async () => {
    await render(<DocBody body={richBody} />);

    expect(await screen.findByText("Title")).toBeTruthy();
    expect(await screen.findByText("First item")).toBeTruthy();
    expect(await screen.findByText('fmt.Println("hi")')).toBeTruthy();
    expect(await screen.findByText("Spec")).toBeTruthy();
    expect(await screen.findByLabelText("Diagram")).toBeTruthy();
  });

  test("a link to another doc opens that doc's screen instead of the system browser", async () => {
    await render(<DocBody body={richBody} />);

    await userEvent.setup().press(await screen.findByText("the other doc"));

    expect(mockPush).toHaveBeenCalledWith("/more/docs/doc-2");
  });

  test("falls back to plain text for a non-JSON legacy body", async () => {
    await render(<DocBody body="Just plain text" />);

    expect(await screen.findByText("Just plain text")).toBeTruthy();
  });
});
