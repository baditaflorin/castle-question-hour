# 0007 — Data pipeline

- Status: not applicable (Mode C)
- Date: 2026-05-11

## Decision

Mode C has no offline data-generation pipeline. Question bank is a committed JSON file
([backend/configs/questions.json](../../backend/configs/questions.json)) loaded at startup.

Operator can edit the JSON and re-deploy; that's the entire "data pipeline".

If we add curated theme dictionaries or sample-castle replay data later, we'll write 0007 then.
