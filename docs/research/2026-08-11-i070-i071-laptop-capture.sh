#!/usr/bin/env bash
# I070/I071 work-laptop capture — one shot.
#
# Run ON THE WORK LAPTOP (the machine with claude-auto + custom gateway):
#
#   bash docs/research/2026-08-11-i070-i071-laptop-capture.sh
#
# It pauses ONCE, mid-run, for you to run your normal claude-auto invocation
# in a throwaway directory with a prompt it gives you. Everything else is
# automatic. Output: ONE tarball in $HOME. Review it for secrets (especially
# claude-auto.source.txt) before bringing it back — redaction is best-effort.
#
# What it collects:
#   I071 — claude-auto identity (path/type/symlink), --help/--version grammar,
#          whether it wraps stock claude (source read), env var NAMES,
#          gateway endpoint HOST (never token values), settings/model-override
#          surfaces incl. managed settings.
#   I070 — controller + subagent session .jsonl from a live gateway run,
#          per-turn message.model extraction, agent meta toolUseId correlation,
#          your declared model/effort/gateway version (asked interactively).

set -u

TS=$(date +%Y%m%d-%H%M%S)
OUT="$HOME/i070-i071-capture-$TS"
mkdir -p "$OUT"
C="$OUT/commands.txt"
: >"$C"

say() { printf '\n==> %s\n' "$*"; }
rec() { # rec <cmd...> — record command + output into commands.txt, never abort
  printf '$ %s\n' "$*" >>"$C"
  "$@" >>"$C" 2>&1
  printf '\n' >>"$C"
}
redact() {
  perl -pe 's/((?:api[_-]?key|auth[_-]?token|token|secret|password|bearer)["'"'"'\s]*[=:]\s*)\S+/${1}REDACTED/gi;
            s/\bbearer\s+\S+/bearer REDACTED/gi;
            s/\bsk-[A-Za-z0-9_-]{8,}/sk-REDACTED/g'
}
ask() { # ask <question> — append Q/A to owner-answers.txt
  printf '%s ' "$1"
  IFS= read -r reply
  printf 'Q: %s\nA: %s\n\n' "$1" "$reply" >>"$OUT/owner-answers.txt"
}

# ---------- Section A: claude-auto identity & grammar (I071) ----------
say "A: claude-auto identity & grammar"
CA=$(command -v claude-auto || true)
if [ -z "$CA" ]; then
  echo "claude-auto NOT on PATH — record where it actually lives:" >>"$C"
  ask "claude-auto is not on PATH here — what is its full path (or 'unknown')?"
  CA=$(tail -1 "$OUT/owner-answers.txt" | sed 's/^A: //')
fi
rec command -v claude-auto
[ -n "$CA" ] && [ -e "$CA" ] && {
  rec ls -l "$CA"
  rec file "$CA"
  rec readlink "$CA"
  # If it's a text script, capture redacted source — this answers "does it
  # delegate to stock claude", arg forwarding, env handling in one artifact.
  if file "$CA" | grep -qi 'text'; then
    redact <"$CA" >"$OUT/claude-auto.source.txt"
    echo "captured redacted source -> claude-auto.source.txt" >>"$C"
  else
    echo "claude-auto is a binary; source not capturable" >>"$C"
  fi
}
rec claude-auto --version
rec claude-auto --help
rec claude-auto -h
rec command -v claude
rec claude --version
CL=$(command -v claude || true)
[ -n "$CL" ] && { rec ls -l "$CL"; rec file "$CL"; }

# ---------- Section B: env & settings surfaces (I071) ----------
say "B: env names, endpoint host, settings/model-override surfaces"
{ printf '$ env var NAMES matching provider/claude patterns\n'
  printenv | cut -d= -f1 | grep -E '^(ANTHROPIC|CLAUDE|AWS_|VERTEX|GOOGLE_|OPENAI)' | sort
  printf '\n'
} >>"$C" 2>&1
# Endpoint host only — scheme://host[:port], never path/query/credentials.
if [ -n "${ANTHROPIC_BASE_URL:-}" ]; then
  printf 'ANTHROPIC_BASE_URL host: %s\n\n' \
    "$(printf '%s' "$ANTHROPIC_BASE_URL" | awk -F/ '{print $1"//"$3}')" >>"$C"
else
  printf 'ANTHROPIC_BASE_URL: unset in this shell\n\n' >>"$C"
fi
for f in "$HOME/.claude/settings.json" "$HOME/.claude/settings.local.json" \
         "/Library/Application Support/ClaudeCode/managed-settings.json"; do
  if [ -f "$f" ] && command -v jq >/dev/null 2>&1; then
    printf '$ settings surface: %s\n' "$f" >>"$C"
    jq -r '"model=" + (.model // "<absent>"),
           "effortLevel=" + (.effortLevel // "<absent>"),
           "env.keys=" + (if has("env") then (.env | keys | join(",")) else "<absent>" end),
           "modelOverrides=" + (if has("modelOverrides") then (.modelOverrides | tostring) else "<absent>" end),
           "availableModels=" + (if has("availableModels") then (.availableModels | tostring) else "<absent>" end),
           "enforceAvailableModels=" + (if has("enforceAvailableModels") then (.enforceAvailableModels | tostring) else "<absent>" end)' \
      "$f" >>"$C" 2>&1
    printf '\n' >>"$C"
  else
    printf 'settings surface absent/unreadable: %s\n\n' "$f" >>"$C"
  fi
done

# ---------- Section C: live gateway run (I070) ----------
say "C: live capture"
CAP_DIR=$(mktemp -d /tmp/i070-cap.XXXXXX)
touch "$OUT/.marker"
cat >"$OUT/prompt.txt" <<'EOF'
Use the Task tool to dispatch one subagent with exactly this prompt:
"Reply exactly: I070-laptop-capture-OK". After the subagent returns,
reply exactly: DONE
EOF
say "In ANOTHER terminal:"
printf '   1. cd %s\n' "$CAP_DIR"
printf '   2. Run your NORMAL day-to-day claude-auto invocation targeting a\n'
printf '      3rd-party model through the gateway (interactive or -p, either\n'
printf '      is fine), giving it EXACTLY this prompt:\n\n'
cat "$OUT/prompt.txt"
printf '\n   3. Wait for DONE, exit claude, come back here.\n\n'
printf 'Press Enter when the run is complete... '
IFS= read -r _

# The grammar we need, from the person who knows it (redact any token):
ask "Exact command you ran (redact tokens):"
ask "Declared/intended model id for that run:"
ask "Requested effort, if any (flag/env/wrapper default):"
ask "Gateway name + version, if known:"
ask "Does claude-auto wrap/exec stock 'claude'? (yes/no/unknown + detail):"
ask "Where do the gateway endpoint + auth live? (env/config/keychain — names only):"

# ---------- Section D: harvest transcripts ----------
say "D: harvesting transcripts"
esc=$(printf '%s' "$CAP_DIR" | tr '/.' '--')
PROJ="$HOME/.claude/projects/$esc"
if [ ! -d "$PROJ" ]; then
  # Fallback: newest project dir since the marker (escaping rules may drift)
  PROJ=$(find "$HOME/.claude/projects" -maxdepth 1 -type d -newer "$OUT/.marker" | head -1)
  printf 'escaped dir not found; marker fallback chose: %s\n\n' "${PROJ:-none}" >>"$C"
fi
if [ -n "${PROJ:-}" ] && [ -d "$PROJ" ]; then
  cp -R "$PROJ" "$OUT/transcripts"
  rec ls -laR "$PROJ"
  if command -v jq >/dev/null 2>&1; then
    for j in "$PROJ"/*.jsonl "$PROJ"/*/subagents/agent-*.jsonl; do
      [ -f "$j" ] || continue
      printf '$ per-turn models: %s\n' "$j" >>"$C"
      jq -c 'select(.type == "assistant") |
             {model: .message.model, content_types: [.message.content[].type]}' \
        "$j" >>"$C" 2>&1
      printf '\n' >>"$C"
    done
    for m in "$PROJ"/*/subagents/agent-*.meta.json; do
      [ -f "$m" ] || continue
      printf '$ subagent meta: %s\n' "$m" >>"$C"
      jq -c '{toolUseId: (.toolUseId // "<absent>")}' "$m" >>"$C" 2>&1
      printf '\n' >>"$C"
    done
  else
    echo "jq missing — raw transcripts copied, extraction skipped" >>"$C"
  fi
else
  echo "NO transcript dir found — record the run dir + session id by hand" >>"$C"
  ask "Transcript harvest failed — where did the session land, if you know?"
fi

# ---------- Package ----------
rm -f "$OUT/.marker"
TARBALL="$HOME/i070-i071-capture-$TS.tgz"
tar -czf "$TARBALL" -C "$HOME" "i070-i071-capture-$TS"
say "DONE. Tarball: $TARBALL"
say "REVIEW before sharing: claude-auto.source.txt and owner-answers.txt for anything the redactor missed."
say "Bring the tarball back to the spine session (drop it anywhere readable, e.g. ~/Downloads on the main machine)."
