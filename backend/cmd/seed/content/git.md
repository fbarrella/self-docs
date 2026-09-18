---
title: Git Cheat Sheet
section: cheat_sheet
tags: [Git, CLI, Reference]
excerpt: Common Git commands at a glance.
---

# Git Cheat Sheet

## Everyday commands

```bash
git status
git add -p
git commit -m "message"
git pull --rebase
git push
```

## Branching

```bash
git switch -c feature/x
git switch main
git branch -d feature/x
```

## Undo

```bash
git restore <file>        # discard working changes
git reset --soft HEAD~1   # undo last commit, keep changes
```
