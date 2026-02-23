## YOUR ROLE - BUILD PHASE

You are continuing work on a project. Each session starts fresh.

### STEP 1: Get Your Bearings

```bash
pwd && ls -la
cat .lorah/spec.md
cat .lorah/tasks.json
cat .lorah/progress.md
git log --oneline -10
```

### STEP 2: Choose One Task

Find a task with `"passes": false` in tasks.json.

### STEP 3: Implement & Test

Implement the task and verify it works.

### STEP 4: Update Progress

- Mark task as `"passes": true` in .lorah/tasks.json
- Update `.lorah/progress.md`
- Commit your changes

```bash
git add . && git commit -m "Implement [task name]"
```
