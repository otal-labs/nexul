# 14 — Read-only harness readiness endpoint

**What to build:** The web can ask "can this user run a play against this project right now" without starting a turn. A gateway route runs the existing pairing resolution for the caller and a project and answers either the resolved computer id or one of the four existing not-configured reasons; the web joins the computer id against the presence poll to know "offline". A hook exposes readiness as ready, unpaired, expired, no harness project, several computers with no default, or offline, with the copy for each. The Pairing settings page shows the same readiness line so the reasons are visible before any button exists.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] Route answers with `ok` and the computer id, or `ok:false` and the reason, for the caller only; never leaks another user's pairing
- [ ] A harness that is paired but not connected reads as offline in the web
- [ ] Copy says harness, never computer: "Pair a harness in Settings to run plays", "Your harness is offline", and one line per remaining reason
- [ ] Tests cover every reason and the offline join
