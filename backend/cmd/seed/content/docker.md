---
title: Docker Commands
section: cheat_sheet
tags: [Docker, CLI, Reference]
excerpt: Container and Compose quick reference.
---

# Docker Commands

```bash
docker ps -a
docker logs -f <container>
docker exec -it <container> sh
docker compose up --build -d
docker compose down -v
```

## Cleaning up

```bash
docker system prune -af --volumes
```
