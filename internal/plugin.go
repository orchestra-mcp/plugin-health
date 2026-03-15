// Package internal contains the core registration logic for the tools.health
// plugin. The HealthPlugin struct wires all 14 tool handlers to the plugin
// builder with their schemas and descriptions.
package internal

import (
	"github.com/orchestra-mcp/plugin-health/internal/storage"
	"github.com/orchestra-mcp/plugin-health/internal/tools"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// HealthPlugin holds the storage reference and registers all tools.
type HealthPlugin struct {
	Storage *storage.DataStorage
}

// RegisterTools registers all 14 health management tools with the plugin builder.
func (hp *HealthPlugin) RegisterTools(builder *plugin.PluginBuilder) {
	s := hp.Storage

	// CRUD tools (8)
	builder.RegisterTool("log_water",
		"Log water intake with amount in ml and optional timestamp",
		tools.LogWaterSchema(), tools.LogWater(s))

	builder.RegisterTool("log_meal",
		"Log a meal with name, type (breakfast/lunch/dinner/snack), calories, and optional notes",
		tools.LogMealSchema(), tools.LogMeal(s))

	builder.RegisterTool("log_caffeine",
		"Log caffeine intake with type (coffee/tea/energy_drink), amount in mg, and optional timestamp",
		tools.LogCaffeineSchema(), tools.LogCaffeine(s))

	builder.RegisterTool("start_pomodoro",
		"Start a pomodoro work session with optional task description and duration",
		tools.StartPomodoroSchema(), tools.StartPomodoro(s))

	builder.RegisterTool("end_pomodoro",
		"End a pomodoro session by ID with optional notes about what was accomplished",
		tools.EndPomodoroSchema(), tools.EndPomodoro(s))

	builder.RegisterTool("update_health_profile",
		"Update health profile with weight, conditions, wake/sleep times, water target, and caffeine config",
		tools.UpdateHealthProfileSchema(), tools.UpdateHealthProfile(s))

	builder.RegisterTool("get_health_profile",
		"Get the current health profile with all settings and conditions",
		tools.GetHealthProfileSchema(), tools.GetHealthProfile(s))

	builder.RegisterTool("start_shutdown",
		"Start the evening shutdown routine (GERD protocol, screen reduction, sleep prep)",
		tools.StartShutdownSchema(), tools.StartShutdown(s))

	// Intelligence tools (6)
	builder.RegisterTool("get_hydration_status",
		"Calculate current hydration status: total consumed, target, deficit, percentage, and hourly rate needed",
		tools.GetHydrationStatusSchema(), tools.GetHydrationStatus(s))

	builder.RegisterTool("nutrition_safety_check",
		"Check if a food is safe given health conditions (gout, GERD, IBS, fatty liver)",
		tools.NutritionSafetyCheckSchema(), tools.NutritionSafetyCheck(s))

	builder.RegisterTool("get_shutdown_status",
		"Get evening shutdown status: minutes until shutdown, whether active, and compliance",
		tools.GetShutdownStatusSchema(), tools.GetShutdownStatus(s))

	builder.RegisterTool("get_caffeine_window",
		"Check if current time is within the safe caffeine window based on cortisol cycle",
		tools.GetCaffeineWindowSchema(), tools.GetCaffeineWindow(s))

	builder.RegisterTool("get_health_summary",
		"Get a comprehensive health summary for today with all protocol scores",
		tools.GetHealthSummarySchema(), tools.GetHealthSummary(s))

	builder.RegisterTool("health_reminder",
		"Get contextual, actionable health reminders based on current time and health state",
		tools.HealthReminderSchema(), tools.HealthReminder(s))
}
