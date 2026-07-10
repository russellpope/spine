# Workflow — mixed

profile: go-service
template_version: 6
model_routing:
  primary: claude-fable-5          # design, plan, implement, orchestrate
  routine: claude-sonnet-5         # routine subagent roles
  mechanical: claude-haiku-4-5     # verbatim plan transcription only
  fallback: claude-opus-4-8        # security-framed / refusal fallback
  notes: unknown keys are ignored  # parser tolerance
effort: high
stages: [grill, prd, issues, implement, review, verify, ship]
