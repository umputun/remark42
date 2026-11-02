---
worth: yes
where: repository settings, not a file
added: 2026-08-21
---
# nothing on master is a required status check

`master` is protected, but `required_status_checks` is `{"checks": [], "contexts": [], "enforcement_level": "off"}`,
and the only ruleset on the branch ("Copilot review for default branch", id 15225632) carries just
`deletion`, `non_fast_forward` and `copilot_code_review`. Verified with
`gh api repos/umputun/remark42/branches/master`.

Every workflow in the repository is therefore advisory. A red X on any backend, frontend or site job
shows in the checks list and does not stop the merge button, so a pull request that fails CI can still
be merged by anyone who does not read the list.

Fix: configure required status checks on `master` for the jobs that should block. Worth deciding
deliberately rather than by default, since it also governs what happens to Dependabot pull requests
and to any job that turns out to be flaky.
