# 21 — Per-play user exclusion

**What to build:** The permission overwrite table gains the `play` resource type, and the doc sharing dialog is generalised to work for any resource type so a play's settings page can deny `plays:run` to named users. Listing the plays that apply to a ticket or doc, on the page and through MCP, honours both the user denial and the play's excluded projects.

**Blocked by:** 20

**Status:** done

- [ ] From a play's settings page, denying a user hides that play from them everywhere and `play_list` omits it for them
- [ ] A play excluded from a project never lists for that project's tickets or docs
- [ ] The docs sharing dialog keeps working unchanged
- [ ] The Owner bypass ignores the denial
