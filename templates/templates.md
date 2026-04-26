# 常用模板合集

## Item Shard Template

```yaml
id: item.<id>
type: item
name:
aliases: []
canonical_status: active
summary:
first_seen:
  chapter:
  scene:
  anchor:
  summary:
  evidence_level:
latest_seen:
  chapter:
  scene:
  anchor:
  summary:
  evidence_level:
current_owner:
  holder:
  confidence:
  source:
appearances:
  - chapter:
    anchor:
    role:
    event:
    facts: []
current_canon:
  confirmed: []
  unconfirmed: []
usage_rules:
  can_write: []
  cannot_write: []
related:
  plot_threads: []
  locations: []
  characters: []
retrieval_policy:
  default_strategy:
  raw_text_required: true
```

## Character Shard Template

```yaml
id: character.<id>
type: character
name:
aliases: []
role:
first_seen:
  chapter:
  anchor:
latest_seen:
  chapter:
  anchor:
current_state:
  location:
  goal:
  resources: []
  vulnerability: []
knowledge_state:
  knows: []
  suspects: []
  does_not_know: []
voice_profile:
  typical: []
  when_panicked: []
  forbidden: []
key_appearances:
  - chapter:
    anchor:
    role:
    summary:
retrieval_policy:
  current_state: recent_first
  voice: representative_sample
  origin: origin_first
```

## Fact Card Template

```yaml
fact_card:
  id:
  query:
  answer:
  confidence: high | medium | low
  evidence:
    - source_type: raw_chapter | snapshot | ledger | shard | inference
      chapter:
      anchor:
      summary:
      evidence_level:
  allowed_usage: []
  forbidden_usage: []
  unresolved: []
  conflicts: []
  followup_queries: []
```

## Context Pack Template

```markdown
# Task
# Reader Contract
# Chapter Intent
# Scene Cards
# Must Carry Forward
# Must Not Contradict
# Active Characters
# Info Gap
# Plot Threads
# Reader Promise / Payoff Ledger
# Relevant Evidence Cards
# Relevant Raw Text Windows
# Style Constraints
# Output Contract
# Included Sources
# Uncertainty
```

## Retrieval Plan Template

```yaml
query:
query_type:
strategy:
  primary:
  secondary: []
targets:
  entities: []
  aliases: []
sources_to_check:
  - shard_index
  - chapter_snapshots
  - raw_chapter_text
search_steps:
  - step:
    tool:
    input:
expected_output:
  - exact_fact
  - source_chapter
  - evidence_excerpt
  - confidence
  - allowed_usage
  - forbidden_usage
```
