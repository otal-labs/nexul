// @vitest-environment node
import { ESLint } from "eslint";
import { describe, expect, it } from "vitest";

// Vitest runs from web/, so ESLint finds eslint.config.js there and lints the probe as a file under src/.
const lint = async (code: string) => {
  const [result] = await new ESLint().lintText(code, { filePath: "src/components/Probe.tsx" });
  return result!.messages.filter((m) => m.message.includes("OverflowCount")).map((m) => m.line);
};

describe("the title-list lint rule", () => {
  it("fails a DOM element whose title joins a list, and passes a component's title prop", async () => {
    const code = [
      'export const Hidden = ({ names }: { names: string[] }) => <span title={names.join(", ")}>+{names.length}</span>;',
      'export const Mapped = ({ names }: { names: string[] }) => <p title={names.map((n) => n.trim()).join(", ")}>x</p>;',
      'const Notice = ({ title }: { title: string }) => <h3>{title}</h3>;',
      'export const Heading = ({ names }: { names: string[] }) => <Notice title={`${names.join(", ")} changed`} />;',
      'export const Single = ({ name }: { name: string }) => <span title={name}>{name}</span>;',
    ].join("\n");

    expect(await lint(code)).toEqual([1, 2]);
  });
});
