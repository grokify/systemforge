# AGENTS Directory

This directory contains pre-computed analysis and context for AI coding agents.

## Contents

| File | Purpose | Freshness |
|------|---------|-----------|
| `GRAPH_SUMMARY.md` | Lightweight graph statistics and key nodes | Check commit date |
| `ARCHITECTURE.md` | High-level architecture description | Manual updates |

## For Agents

If you're an AI agent working on this codebase:

1. **Start here** - Read `GRAPH_SUMMARY.md` for codebase structure overview
2. **Key hubs** - The "Key Nodes" section shows central files/functions
3. **Regenerate if stale** - If this data is outdated, run:
   ```bash
   graphize analyze && graphize summary -o AGENTS/GRAPH_SUMMARY.md
   ```

## For Humans

To regenerate the graph analysis:

```bash
# Install graphize (if not already)
go install github.com/plexusone/graphize/cmd/graphize@latest

# Analyze and generate summary
cd /path/to/repo
graphize init
graphize add .
graphize analyze
graphize summary -o AGENTS/GRAPH_SUMMARY.md

# Optional: Generate interactive HTML (not checked in due to size)
graphize export html -o /tmp/graph.html && open /tmp/graph.html
```

## Why Not Check In the Full Graph?

| What | Size | Recommendation |
|------|------|----------------|
| `GRAPH_SUMMARY.md` | ~3KB | Check in |
| `graph.json` | ~30MB | Don't check in |
| `graph.html` | ~25MB | Don't check in |
| `.graphize/` database | ~50MB | Add to `.gitignore` |

Large files bloat git history. Generate them on-demand instead.

## Staleness

The graph summary was last generated from commit: `<!-- UPDATE THIS -->`

If the codebase has changed significantly, regenerate the summary.
