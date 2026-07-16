---
id: I011
title: Blocking home for the stage check
severity: med
status: fixed
affects: [audit, doctor]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

Where does the mechanical stage check live, and which invocation blocks — `spine doctor` (as the destination originally read) or a new `spine audit stages`? Requirements-attack note: doctor is documented as read-only advisory health checks; blocking enforcement today lives in `spine audit` (routing), so "doctor fails" contradicted the existing architecture.

## Resolution

(2026-07-15, owner) New **`spine audit stages`** exits non-zero on cursor/artifact mismatch and joins `spine audit routing` in the verify gate; **doctor gains an advisory D-check** using the same shared derivation engine. Doctor stays advisory; hooks and /handoff get one blocking command to call. The map's destination wording is amended accordingly.
