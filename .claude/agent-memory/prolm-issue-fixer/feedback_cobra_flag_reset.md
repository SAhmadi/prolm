---
name: Cobra flag reset between test invocations
description: Cobra persists flag Changed state across Execute() calls in the same test binary; must call ResetFlags and re-register flags in test helpers.
type: feedback
---

When multiple tests call `rootCmd.SetArgs(...)` and `Execute()` within the same binary run, Cobra's `FlagSet` retains the `Changed` state from the previous call. A flag set in one test (e.g. `--frozen`) will still appear as `Changed` in the next test's `RunE`, causing spurious errors.

**Why:** Cobra's `pflag.FlagSet` is mutable and not reset between `Execute()` calls unless explicitly cleared.

**How to apply:** In test setup helpers (e.g. `resetRootCmd`), call `cmd.ResetFlags()` on any subcommand whose flags were touched, then re-register the flags manually. This is the pattern used in `cmd/install_test.go` after QUALITY-017.
