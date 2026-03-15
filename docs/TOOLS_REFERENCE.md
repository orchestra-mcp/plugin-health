# Tools Reference

## CRUD Tools

### log_water

Log a water intake event.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `amount_ml` | integer | Yes | Amount of water in milliliters |
| `is_gout_flush` | boolean | No | Whether this is a gout protocol flush (default: false) |

### log_meal

Log a meal with safety classification.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Meal name/description |
| `is_safe` | boolean | Yes | Whether the meal is safe for user's conditions |
| `category` | string | No | Meal category (default: "other") |
| `triggers` | string[] | No | Condition-specific trigger warnings |

### log_caffeine

Log a caffeine intake event.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `drink_type` | string | Yes | Type of drink (e.g., "black_coffee", "red_bull") |
| `is_clean` | boolean | Yes | Whether the drink is sugar-free/clean |
| `caffeine_mg` | integer | No | Caffeine content in mg (default: 0) |
| `sugar_g` | number | No | Sugar content in grams (default: 0) |

### start_pomodoro

Start a new pomodoro work/break/stand session.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `type` | string | No | Session type: "work", "break", or "stand" (default: "work") |
| `duration_min` | integer | No | Session duration in minutes (default: 25) |

### end_pomodoro

End an active pomodoro session.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | Yes | Session ID to end |
| `completed` | boolean | No | Whether the session was completed (default: true) |
| `stood_up` | boolean | No | Whether the user stood up during break (default: false) |
| `walked_min` | number | No | Minutes walked during break (default: 0) |

### update_health_profile

Update health configuration and targets.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `target_water_ml` | integer | No | Daily water target in ml |
| `work_window_start` | string | No | Work window start (HH:MM) |
| `work_window_end` | string | No | Work window end (HH:MM) |
| `target_sleep_time` | string | No | Target bedtime (HH:MM) |
| `pomodoro_duration_min` | integer | No | Pomodoro work duration |
| `pomodoro_daily_target` | integer | No | Daily pomodoro target |
| `conditions` | string[] | No | Health conditions (gout, gerd, ibs, fatty_liver, sinusitis) |

### get_health_profile

Get the current health profile. No arguments required.

### start_shutdown

Activate GERD shutdown mode. No arguments required.

---

## Intelligence Tools

### get_hydration_status

Get current hydration progress, deficit, and hourly rate needed to meet target. No arguments required.

**Response includes:** current_ml, target_ml, deficit_ml, percentage, hourly_rate_needed, on_track

### nutrition_safety_check

Check if a food is safe for the user's health conditions.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `food` | string | Yes | Food name to check |

**Response includes:** is_safe, triggers, alternatives, explanation

### get_shutdown_status

Get GERD shutdown window status and allowed items. No arguments required.

**Response includes:** active, time_remaining, allowed_items, last_meal_gap_hours

### get_caffeine_window

Check cortisol window compliance and caffeine transition score. No arguments required.

**Response includes:** in_cortisol_window, cortisol_window_end, transition_score, clean_count, sugar_count, recommendation

### get_health_summary

Get daily aggregate of all 5 health protocols. No arguments required.

**Response includes:** hydration, nutrition, caffeine, pomodoro, shutdown scores + overall_score (0-100)

### health_reminder

Get a context-aware health nudge based on current time and deficits. No arguments required.

**Response includes:** type (water/stand/shutdown/caffeine/nutrition), message, priority, action
