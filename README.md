# Orchestra Plugin: plugin-health

Health management tools for the [Orchestra MCP](https://github.com/orchestra-mcp/framework) framework. Powered by **Health Debug**.

Tracks 5 health protocols for software engineers: hydration, pomodoro/standing, GERD shutdown, caffeine transition, and nutrition safety.

## Install

```bash
go install github.com/orchestra-mcp/plugin-health/cmd@latest
```

## Usage

Add to your `plugins.yaml`:

```yaml
- id: tools.health
  binary: ./bin/plugin-health
  enabled: true
```

## Tools

### CRUD Tools (8)

| Tool | Description |
|------|-------------|
| `log_water` | Log water intake (amount_ml, is_gout_flush) |
| `log_meal` | Log meal (name, is_safe, category, triggers) |
| `log_caffeine` | Log caffeine (drink_type, is_clean, caffeine_mg, sugar_g) |
| `start_pomodoro` | Start work/break/stand session |
| `end_pomodoro` | End session (completed, stood_up, walked_min) |
| `update_health_profile` | Update health config (targets, work window, conditions) |
| `get_health_profile` | Get current health profile |
| `start_shutdown` | Activate GERD shutdown mode |

### Intelligence Tools (6)

| Tool | Description |
|------|-------------|
| `get_hydration_status` | Current water progress, deficit, hourly rate needed |
| `nutrition_safety_check` | Check if a food is safe for user's conditions |
| `get_shutdown_status` | GERD shutdown window status, allowed items |
| `get_caffeine_window` | Cortisol window check, clean vs sugar advice |
| `get_health_summary` | Daily aggregate of all 5 protocols |
| `health_reminder` | Context-aware nudge based on time + deficits |

## Related Packages

- [sdk-go](https://github.com/orchestra-mcp/sdk-go) — Plugin SDK
- [gen-go](https://github.com/orchestra-mcp/gen-go) — Generated Protobuf types
