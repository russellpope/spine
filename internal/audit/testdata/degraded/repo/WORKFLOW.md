# Workflow — degraded

profile: go-service
template_version: 5
model_routing:
  primary: claude-fable-5          # long-horizon work
  fallback: claude-opus-4-8        # refusal fallback
  routine: claude-sonnet-5         # routine subagent roles
effort: high
