// @vitest-environment node
import { describe, expect, it } from "vitest";

import { previewKind, previewMaxBytes } from "@/models/Attachment";

describe("previewKind", () => {
  it.each([
    ["a raster image", "shot.png", "image/png", "image"],
    ["an SVG the server sniffed as XML", "diagram.svg", "text/xml; charset=utf-8", "svg"],
    ["an SVG the server sniffed as plain text", "diagram.svg", "text/plain; charset=utf-8", "svg"],
    ["a PDF", "spec.pdf", "application/pdf", "pdf"],
    ["an HTML page named like a PDF", "spec.pdf", "text/html; charset=utf-8", "none"],
    ["an MP3", "voice.mp3", "audio/mpeg", "audio"],
    ["an M4A the server sniffed as video", "voice.m4a", "video/mp4", "audio"],
    ["a FLAC the server could not sniff", "take.flac", "application/octet-stream", "audio"],
    ["an HTML page named like audio", "voice.mp3", "text/html; charset=utf-8", "none"],
    ["a CSV", "rows.csv", "text/plain; charset=utf-8", "csv"],
    ["a TSV", "rows.tsv", "text/plain; charset=utf-8", "csv"],
    ["a binary file named like a CSV", "rows.csv", "application/octet-stream", "none"],
    ["a SQLite database", "dump.sqlite", "application/octet-stream", "none"],
    ["a file with no extension", "README", "text/plain; charset=utf-8", "none"],
  ])("reads %s", (_, name, contentType, kind) => {
    expect(previewKind({ name, content_type: contentType, size: 1024 })).toBe(kind);
  });

  it("downloads a previewable file past the limit instead of loading it", () => {
    expect(previewKind({ name: "spec.pdf", content_type: "application/pdf", size: previewMaxBytes + 1 })).toBe("too-large");
    expect(previewKind({ name: "dump.zip", content_type: "application/zip", size: previewMaxBytes + 1 })).toBe("none");
  });
});
