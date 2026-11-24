# How I Attribute AI Work

When AI makes a significant contribution to a change, I say so. This
applies to commits, pull requests, review comments and other work that
I publish.

## Why

- Reviewers and readers do not have to ask if AI wrote the change. The
  answer is always in the change.
- I hope that it encourages other people to do the same. When this
  disclosure is normal, it is easier for teams to talk openly about how
  they use AI.

AI attribution does not change my ownership. I review and approve all
work that has my name on it, and I am responsible for it.

## How

An agent instruction adds the attribution automatically. My agent
configuration tells each AI agent to add a marker to its work, for
example:

- A pull request description ends with `🤖 Authored using AI (<model>)`.
- A review comment reply ends with `🤖 Authored by AI (<model>)`.
- A commit includes a `Co-authored-by` trailer that names the model.

Each marker links back to this page with a permalink to its latest
version. The [ai-attribution-link](bin/ai-attribution-link) script gets
that permalink. Refer to the
[Pull Requests section of AGENTS.md](AGENTS.md#pull-requests) for the
exact instructions.
