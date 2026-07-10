# Workflow — clean

profile: go-service
template_version: 5
model_routing:
  primary: claude-fable-5          # long-horizon, ambiguous, or first-shot-complex work
  fallback: claude-opus-4-8        # security-framed / refusal fallback
  routine: claude-sonnet-5         # mechanical subagent roles
effort: high
stages: [grill, prd, issues, implement, review, verify, ship]
