# 02 — Stored per-device sessions

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

What exactly replaces the stateless 24-hour session token? The table and its columns, how platform and client are recorded (user agent for browsers, reported by the desktop and phone apps), the sliding-expiry rules, how the per-request check folds into the existing user reload, what happens to tokens issued before the change, and which surfaces it reaches: the HTTP routes for listing and signing out devices, whether agents get an MCP tool for it (list only, or none), and the events a sign-out publishes. Ends with the ADR that supersedes ADR 0041.
