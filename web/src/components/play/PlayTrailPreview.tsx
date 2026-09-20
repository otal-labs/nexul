// Decorative terminal mockup used as HomePage's hero visual; extracted out since it isn't page-specific (F7).
export const PlayTrailPreview = () => (
  <div className="terminal-window w-full max-w-2xl text-left" aria-hidden>
    <div className="terminal-window__bar">
      <span className="terminal-window__dot" />
      <span className="terminal-window__dot" />
      <span className="terminal-window__dot" />
      <span className="terminal-window__title">nexul · agent trail</span>
    </div>
    <div className="terminal-window__body">
      <p>
        <span className="text-primary">▶</span> Play "Fix with AI" on{" "}
        <span className="text-info">ticket NEX-142</span>
      </p>
      <p className="text-muted-foreground">
        <span className="text-primary">▸</span> read the failing test, found an off-by-one in
        scheduler.go
      </p>
      <p className="text-muted-foreground">
        <span className="text-primary">▸</span> patched scheduler.go, ran go test ./...
      </p>
      <p>
        <span className="text-success">✓</span> done in 48s — trail saved, ticket moved to Review
      </p>
    </div>
  </div>
);
