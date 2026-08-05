# Draft: Lorah vs Interactive Agent Sessions

> **Status:** Temporary notes from a design conversation (2026-08-05).  
> Not yet part of the published guides. Intended to feed a future guide (or README section) on when to use Lorah, when to stay interactive, and optional multi-backend support (e.g. Grok Build).

---

## Context

Lorah is a thin infinite-loop harness for coding agents (today: Claude Code CLI), following the Ralph technique. Its job is loop + readable stream-JSON output + error recovery. The agent owns workflow via prompt files, git, and project docs.

This note compares that model to **interactive** agent sessions (e.g. Grok Build TUI, Claude Code interactive), and records whether supporting additional backends (Grok Build headless) is worth it.

---

## 1. Should Lorah support Grok Build?

### Short answer

**Yes as a thin backend option — only if you actually run unattended Ralph-style loops with Grok.** Do not turn Lorah into an orchestrator that competes with Grok’s native tools (plan mode, workflows, subagents, execute-plan).

### Why it fits mechanically

Grok already exposes the primitives a Ralph harness needs:

| Need | Grok today |
| ---- | ---------- |
| One-shot headless run | `grok --prompt-file … -p --yolo` |
| Stream JSON for pretty-print | `--output-format streaming-messages-json` |
| Unattended tools | `--always-approve` / `--yolo` |
| Flag passthrough | model, max-turns, sandbox, tools, etc. |

A bare shell loop already works:

```bash
while true; do
  grok --prompt-file PROMPT.md -p --yolo \
    --output-format streaming-messages-json
done
```

That recreates the problem Lorah solves: raw stream-JSON is hard to watch, and you lack shared retry / two-stage Ctrl+C behavior.

### What would be valuable

- Backend switch: `claude` vs `grok`, same loop + formatting philosophy
- Map Grok tool names (`read_file`, `run_terminal_cmd`, `search_replace`, …) in the pretty-printer
- Argv differences (`--prompt-file` + `-p` vs pipe-to-stdin `claude -p`)
- Docs: same `.lorah/` SDD workflow works with either CLI

### What would not be valuable

- Importing Grok orchestration (workflows, subagents, plan mode, execute-plan) into Lorah
- Multi-agent schedulers, session state, task CLIs (already removed in 0.7)

That would fight Lorah’s post-0.3 design: *loop + readable output + recovery; agent owns the workflow*.

### Priority sketch

| Priority | Action |
| -------- | ------ |
| High *if* you run Grok Ralph loops | Backend abstraction + output mapping |
| Medium | Docs: dual-backend SDD usage |
| Low / skip | Competing with interactive agent orchestration |

If unattended Grok loops are rare, skip backend work — Grok’s interactive + headless tooling already covers most interactive and scripted cases. Lorah’s edge is **long unattended iteration with readable live output**, not “another agent runtime.”

---

## 2. When is Lorah better than an interactive session?

Use **Lorah** when the bottleneck is **throughput under a fixed contract**, not judgment.

### Good fit

- Specs and acceptance criteria are already clear (`docs/design/`, `.lorah/plan.md`)
- Work is many small, similar tasks (feature slices, greenfield from a solid design, mechanical migration)
- You want **hours unattended** (overnight / while away)
- You want **fresh context every iteration** (Ralph’s anti–context-rot property)
- Continuity lives in **git + task files**, not chat memory
- You’re willing to run with high autonomy (`bypassPermissions` / `--yolo`) and clean up in review

### Shape of the work

```
human: scope + specs + plan done-criteria
lorah: plan → test → implement → commit → repeat
human: review commits / unblock / re-scope
```

Lorah wins when a human mid-flight would mostly say “keep going” and each unit of work fits one invocation.

---

## 3. When is an interactive session better?

Use **interactive** (Grok Build TUI, Claude Code interactive, etc.) when the bottleneck is **judgment, ambiguity, or steering**.

| Situation | Why interactive |
| --------- | --------------- |
| Scoping / architecture / writing specs | Wrong design is expensive; you need pushback |
| Unfamiliar codebase | Exploration needs branching questions |
| Specs are weak or contradictory | Loop will thrash or ship wrong green tests |
| High-stakes or irreversible ops | Permissions, prod, data, security |
| Debugging a stuck loop (`blocked` tasks, flaky tests) | Needs diagnosis, not another blind iteration |
| Short bursts (&lt; ~1–2 hours) | Setup tax of Ralph isn’t worth it |
| Design *is* the product | UX, API shape, product tradeoffs |
| You want rich agent-native power | Plan mode, subagents, design/execute-plan, live approval |

### Rule of thumb

- **Interactive** = raise the quality ceiling (what to build, how, whether done is right)
- **Lorah** = grind toward a ceiling you’ve already set

If you’d interrupt every 10–15 minutes → interactive.  
If you’d leave it overnight with a clear “done when…” → Lorah.

---

## 4. Recommended interactive workflow for larger work

Two good patterns; both start the same way.

### Shared front matter (always interactive)

1. **Explore** — understand repo, constraints, existing patterns
2. **Design** — foundation + feature specs in `docs/design/` (see `docs/guide/`)
3. **Define done** — concrete acceptance criteria (plan file, checklist, or PR plan)
4. **Pick execution mode** — loop vs stayed-interactive vs hybrid

Do not skip design + done-criteria for large bodies of work. **Spec quality is the ceiling for any agent path.**

### Path A — Interactive-only

Best when you stay available and want control at each phase gate.

1. Plan mode / design pass — approach and file-level plan; approve before code
2. Implement in vertical slices — one feature/subsystem per stretch
3. Hard review gates — after each slice: tests, skim diff, adjust plan
4. Use structure when scale demands it — subagents, stacked PRs, CI babysitting
5. Close with integration — full suite, docs, PR

**Strength:** course-correction is cheap. **Weakness:** you are the clock.

### Path B — Hybrid (usually best for large work)

Human on the ambiguous parts; Lorah on the grind.

```
┌─────────────────────────────────────┐
│ INTERACTIVE                         │
│  • design specs                     │
│  • plan.md acceptance criteria      │
│  • .lorah/ prompts + settings       │
│  • first task by hand (calibrate)   │
└──────────────┬──────────────────────┘
               ▼
┌─────────────────────────────────────┐
│ LORAH (unattended)                  │
│  • plan → test → implement loop     │
│  • one task per iteration           │
│  • commits as progress markers      │
└──────────────┬──────────────────────┘
               ▼
┌─────────────────────────────────────┐
│ INTERACTIVE (periodic / end)        │
│  • review commit trail              │
│  • unblock blocked tasks            │
│  • re-scope plan if done drifted    │
│  • polish, CI, PR                   │
└─────────────────────────────────────┘
```

**Calibration tip:** do the first task interactively (or watch the first 1–2 Lorah iterations). That surfaces prompt/spec bugs before they multiply overnight.

### Path C — Interactive with heavier multi-agent orchestration

If you prefer not to use Lorah for a large effort:

1. Design doc + PR plan
2. Execute plan with isolated worktrees / stacked PRs
3. Human review between PRs
4. CI / review follow-through

That’s **orchestrated multi-agent**, not Ralph. Better when work parallelizes across PRs; worse when you want one dumb loop and file-based continuity with minimal harness.

---

## Cheat sheet

| Question | Prefer |
| -------- | ------ |
| Is “done” objectively checkable from git/tests? | Lorah if yes |
| Will I need to change the plan mid-flight often? | Interactive |
| Multi-hour / overnight capacity free? | Lorah |
| Am I still deciding *what* to build? | Interactive |
| One PR vs weekend feature factory? | Interactive vs Lorah (or hybrid) |
| Do I want plan/subagent/workflow tools *in* the loop? | Interactive (or headless scripts), not Lorah |
| Do I just need readable unattended iterations (Claude or Grok)? | Lorah (or Lorah + multi-backend) |

---

## Bottom line

1. **Grok (or other) backend support** is worth it only if you run Ralph-style unattended loops with that CLI — keep it as a thin backend; preserve radical simplicity.
2. **Lorah** for well-spec’d, long, unattended grind with file/git continuity.
3. **Interactive** for design, ambiguity, recovery, short work, and mid-flight judgment.
4. **For large work:** interactive design + done-criteria → optional calibration → Lorah (or orchestrated multi-PR) for bulk implementation → interactive review/unblock. Specs first, always.

---

## Possible future doc homes

When folding this into published docs, candidates:

| Destination | Content |
| ----------- | ------- |
| `docs/guide/when-to-use.md` (new) | §§2–4: Lorah vs interactive, hybrid workflow |
| `docs/guide/workflow.md` | Cross-link; hybrid path as “recommended for large work” |
| `README.md` | Short “When to use Lorah” blurb |
| `docs/design/` or changelog | Only if multi-backend becomes a real feature |

Related existing guides: [workflow](../guide/workflow.md), [configuration](../guide/configuration.md), [prompts](../guide/prompts.md), [tasks](../guide/tasks.md), [design](../guide/design.md).

---

## Open follow-ups (not decided)

- [ ] Commit to multi-backend, or document “Claude only + DIY shell loop for others”?
- [ ] If multi-backend: design sketch for backend interface without CLI ceremony
- [ ] Whether “when to use” belongs in README vs a dedicated guide
- [ ] Any mention of Grok-specific skills/workflows as *alternatives* to Lorah (without implementing them in Lorah)
