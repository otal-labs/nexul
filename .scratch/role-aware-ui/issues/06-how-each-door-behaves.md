# 06: How each one-way door behaves

Type: grilling
Status: open
Blocked by: 02

## Question

For each door ticket 02 finds, decide the way out and its edge cases. For example:
- Deleting a machine that still has runners or stacks targeting it.
- Deleting a tunnel that gateways or hostnames still use.
- Whether saving a stack's environment redeploys it or waits for the next deploy, and who may read a secret's value
  again.
- Whether retrying a failed deploy rebuilds the same commit or the branch's latest.
- What each confirm says.
