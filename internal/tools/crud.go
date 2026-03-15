package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-health/internal/storage"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// ---------- log_water ----------

// LogWaterSchema returns the JSON Schema for the log_water tool.
func LogWaterSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"amount_ml":  map[string]any{"type": "number", "description": "Amount of water in milliliters"},
			"timestamp":  map[string]any{"type": "string", "description": "ISO 8601 timestamp (optional, defaults to now)"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
		"required": []any{"amount_ml"},
	})
	return s
}

// LogWater returns a tool handler that logs a water intake entry.
func LogWater(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "amount_ml"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		amountML := helpers.GetInt(req.Arguments, "amount_ml")
		timestamp := helpers.GetString(req.Arguments, "timestamp")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		if amountML <= 0 {
			return helpers.ErrorResult("validation_error", "amount_ml must be positive"), nil
		}

		if timestamp == "" {
			timestamp = helpers.NowISO()
		}

		entryID := newEntryID("wtr")

		meta := map[string]any{
			"id":         entryID,
			"type":       "water",
			"amount_ml":  amountML,
			"created_at": timestamp,
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		_, err = store.WriteEntry(ctx, projectID, storage.WaterDir, entryID, metadata, "", 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("Logged **%d ml** water.\n\n- **ID:** %s\n- **Time:** %s",
			amountML, entryID, timestamp)
		return helpers.TextResult(md), nil
	}
}

// ---------- log_meal ----------

// LogMealSchema returns the JSON Schema for the log_meal tool.
func LogMealSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":       map[string]any{"type": "string", "description": "Name of the meal or food item"},
			"meal_type":  map[string]any{"type": "string", "description": "Type: breakfast, lunch, dinner, or snack", "enum": []any{"breakfast", "lunch", "dinner", "snack"}},
			"calories":   map[string]any{"type": "number", "description": "Estimated calories (optional)"},
			"notes":      map[string]any{"type": "string", "description": "Additional notes about the meal (optional)"},
			"timestamp":  map[string]any{"type": "string", "description": "ISO 8601 timestamp (optional, defaults to now)"},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
		"required": []any{"name", "meal_type"},
	})
	return s
}

// LogMeal returns a tool handler that logs a meal entry.
func LogMeal(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "name", "meal_type"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		name := helpers.GetString(req.Arguments, "name")
		mealType := helpers.GetString(req.Arguments, "meal_type")
		calories := helpers.GetInt(req.Arguments, "calories")
		notes := helpers.GetString(req.Arguments, "notes")
		timestamp := helpers.GetString(req.Arguments, "timestamp")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		if timestamp == "" {
			timestamp = helpers.NowISO()
		}

		entryID := newEntryID("meal")

		meta := map[string]any{
			"id":         entryID,
			"type":       "meal",
			"name":       name,
			"meal_type":  mealType,
			"created_at": timestamp,
		}
		if calories > 0 {
			meta["calories"] = calories
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		body := ""
		if notes != "" {
			body = notes
		}

		_, err = store.WriteEntry(ctx, projectID, storage.MealsDir, entryID, metadata, body, 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		calStr := ""
		if calories > 0 {
			calStr = fmt.Sprintf("\n- **Calories:** %d", calories)
		}

		md := fmt.Sprintf("Logged **%s** (%s).\n\n- **ID:** %s%s\n- **Time:** %s",
			name, mealType, entryID, calStr, timestamp)
		return helpers.TextResult(md), nil
	}
}

// ---------- log_caffeine ----------

// LogCaffeineSchema returns the JSON Schema for the log_caffeine tool.
func LogCaffeineSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"caffeine_type": map[string]any{"type": "string", "description": "Type: coffee, tea, or energy_drink", "enum": []any{"coffee", "tea", "energy_drink"}},
			"amount_mg":     map[string]any{"type": "number", "description": "Caffeine amount in milligrams"},
			"timestamp":     map[string]any{"type": "string", "description": "ISO 8601 timestamp (optional, defaults to now)"},
			"project_id":    map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
		"required": []any{"caffeine_type", "amount_mg"},
	})
	return s
}

// LogCaffeine returns a tool handler that logs a caffeine intake entry.
func LogCaffeine(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "caffeine_type", "amount_mg"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		caffeineType := helpers.GetString(req.Arguments, "caffeine_type")
		amountMG := helpers.GetInt(req.Arguments, "amount_mg")
		timestamp := helpers.GetString(req.Arguments, "timestamp")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		if amountMG <= 0 {
			return helpers.ErrorResult("validation_error", "amount_mg must be positive"), nil
		}

		if timestamp == "" {
			timestamp = helpers.NowISO()
		}

		entryID := newEntryID("caf")

		meta := map[string]any{
			"id":            entryID,
			"type":          "caffeine",
			"caffeine_type": caffeineType,
			"amount_mg":     amountMG,
			"created_at":    timestamp,
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		_, err = store.WriteEntry(ctx, projectID, storage.CaffeineDir, entryID, metadata, "", 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("Logged **%d mg** caffeine (%s).\n\n- **ID:** %s\n- **Time:** %s",
			amountMG, caffeineType, entryID, timestamp)
		return helpers.TextResult(md), nil
	}
}

// ---------- start_pomodoro ----------

// StartPomodoroSchema returns the JSON Schema for the start_pomodoro tool.
func StartPomodoroSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task":        map[string]any{"type": "string", "description": "Description of the task to work on (optional)"},
			"duration_min": map[string]any{"type": "number", "description": "Duration in minutes (optional, defaults to 25)"},
			"project_id":  map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// StartPomodoro returns a tool handler that starts a new pomodoro session.
func StartPomodoro(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		task := helpers.GetString(req.Arguments, "task")
		durationMin := helpers.GetInt(req.Arguments, "duration_min")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		if durationMin <= 0 {
			durationMin = 25
		}

		now := helpers.NowISO()
		entryID := newEntryID("pom")

		meta := map[string]any{
			"id":           entryID,
			"type":         "pomodoro",
			"status":       "active",
			"duration_min": durationMin,
			"started_at":   now,
			"created_at":   now,
		}
		if task != "" {
			meta["task"] = task
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		_, err = store.WriteEntry(ctx, projectID, storage.PomodoroDir, entryID, metadata, "", 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		taskStr := ""
		if task != "" {
			taskStr = fmt.Sprintf("\n- **Task:** %s", task)
		}

		md := fmt.Sprintf("Started **%d min** pomodoro.\n\n- **ID:** %s%s\n- **Started:** %s",
			durationMin, entryID, taskStr, now)
		return helpers.TextResult(md), nil
	}
}

// ---------- end_pomodoro ----------

// EndPomodoroSchema returns the JSON Schema for the end_pomodoro tool.
func EndPomodoroSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pomodoro_id": map[string]any{"type": "string", "description": "ID of the pomodoro session to end"},
			"notes":       map[string]any{"type": "string", "description": "Notes about what was accomplished (optional)"},
			"project_id":  map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
		"required": []any{"pomodoro_id"},
	})
	return s
}

// EndPomodoro returns a tool handler that ends a pomodoro session.
func EndPomodoro(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "pomodoro_id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		pomodoroID := helpers.GetString(req.Arguments, "pomodoro_id")
		notes := helpers.GetString(req.Arguments, "notes")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		meta, body, version, err := store.ReadEntry(ctx, projectID, storage.PomodoroDir, pomodoroID)
		if err != nil {
			return helpers.ErrorResult("not_found", fmt.Sprintf("pomodoro %s not found: %v", pomodoroID, err)), nil
		}

		m := meta.AsMap()
		m["status"] = "completed"
		m["ended_at"] = helpers.NowISO()

		if notes != "" {
			body = notes
		}

		updatedMeta, err := structpb.NewStruct(m)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		_, err = store.WriteEntry(ctx, projectID, storage.PomodoroDir, pomodoroID, updatedMeta, body, version)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("Completed pomodoro **%s**.\n\n- **Ended:** %s",
			pomodoroID, helpers.NowISO())
		return helpers.TextResult(md), nil
	}
}

// ---------- update_health_profile ----------

// UpdateHealthProfileSchema returns the JSON Schema for the update_health_profile tool.
func UpdateHealthProfileSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"weight_kg":          map[string]any{"type": "number", "description": "Body weight in kilograms"},
			"wake_time":          map[string]any{"type": "string", "description": "Usual wake time in HH:MM format (UTC)"},
			"sleep_time":         map[string]any{"type": "string", "description": "Target sleep time in HH:MM format (UTC)"},
			"water_target_ml":    map[string]any{"type": "number", "description": "Daily water intake target in milliliters"},
			"caffeine_delay_min": map[string]any{"type": "number", "description": "Minutes after waking to delay caffeine (cortisol window, e.g. 90)"},
			"caffeine_cutoff":    map[string]any{"type": "string", "description": "Latest time for caffeine in HH:MM format (UTC)"},
			"shutdown_before_min": map[string]any{"type": "number", "description": "Minutes before sleep_time to start shutdown routine"},
			"conditions": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Health conditions: gout, gerd, ibs, fatty_liver",
			},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// UpdateHealthProfile returns a tool handler that updates the health profile.
func UpdateHealthProfile(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// Try to read existing profile; start fresh if not found.
		existing := make(map[string]any)
		var version int64
		meta, _, v, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			existing = meta.AsMap()
			version = v
		}

		// Merge fields from arguments.
		if w := helpers.GetFloat64(req.Arguments, "weight_kg"); w > 0 {
			existing["weight_kg"] = w
		}
		if wt := helpers.GetString(req.Arguments, "wake_time"); wt != "" {
			existing["wake_time"] = wt
		}
		if st := helpers.GetString(req.Arguments, "sleep_time"); st != "" {
			existing["sleep_time"] = st
		}
		if wt := helpers.GetInt(req.Arguments, "water_target_ml"); wt > 0 {
			existing["water_target_ml"] = wt
		}
		if cd := helpers.GetInt(req.Arguments, "caffeine_delay_min"); cd > 0 {
			existing["caffeine_delay_min"] = cd
		}
		if cc := helpers.GetString(req.Arguments, "caffeine_cutoff"); cc != "" {
			existing["caffeine_cutoff"] = cc
		}
		if sb := helpers.GetInt(req.Arguments, "shutdown_before_min"); sb > 0 {
			existing["shutdown_before_min"] = sb
		}
		if conditions := helpers.GetStringSlice(req.Arguments, "conditions"); len(conditions) > 0 {
			condValues := make([]any, len(conditions))
			for i, c := range conditions {
				condValues[i] = c
			}
			existing["conditions"] = condValues
		}

		existing["updated_at"] = helpers.NowISO()
		if _, ok := existing["created_at"]; !ok {
			existing["created_at"] = helpers.NowISO()
		}

		updatedMeta, err := structpb.NewStruct(existing)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		_, err = store.WriteProfile(ctx, projectID, updatedMeta, "", version)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		return helpers.TextResult("Health profile updated successfully."), nil
	}
}

// ---------- get_health_profile ----------

// GetHealthProfileSchema returns the JSON Schema for the get_health_profile tool.
func GetHealthProfileSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// GetHealthProfile returns a tool handler that retrieves the health profile.
func GetHealthProfile(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err != nil {
			return helpers.TextResult("No health profile found. Use `update_health_profile` to create one."), nil
		}

		resp, err := helpers.JSONResult(meta.AsMap())
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("format profile: %v", err)), nil
		}
		return resp, nil
	}
}

// ---------- start_shutdown ----------

// StartShutdownSchema returns the JSON Schema for the start_shutdown tool.
func StartShutdownSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// StartShutdown returns a tool handler that starts the evening shutdown routine.
func StartShutdown(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		now := helpers.NowISO()
		entryID := newEntryID("shd")

		meta := map[string]any{
			"id":         entryID,
			"type":       "shutdown",
			"status":     "active",
			"started_at": now,
			"created_at": now,
		}

		metadata, err := structpb.NewStruct(meta)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("build metadata: %v", err)), nil
		}

		checklist := "## Shutdown Checklist\n\n" +
			"- [ ] Stop eating (GERD protocol: 3h before sleep)\n" +
			"- [ ] No caffeine\n" +
			"- [ ] Reduce screen brightness\n" +
			"- [ ] Close work tabs/apps\n" +
			"- [ ] Prepare tomorrow's priorities\n" +
			"- [ ] Light stretching or relaxation\n"

		_, err = store.WriteEntry(ctx, projectID, storage.ShutdownDir, entryID, metadata, checklist, 0)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		md := fmt.Sprintf("Shutdown routine started.\n\n- **ID:** %s\n- **Started:** %s\n\n%s",
			entryID, now, checklist)
		return helpers.TextResult(md), nil
	}
}
