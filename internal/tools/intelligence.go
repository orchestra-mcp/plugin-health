package tools

import (
	"context"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/plugin-health/internal/storage"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// ---------- get_hydration_status ----------

// GetHydrationStatusSchema returns the JSON Schema for the get_hydration_status tool.
func GetHydrationStatusSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// GetHydrationStatus returns a tool handler that calculates current hydration status.
func GetHydrationStatus(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// Load today's water entries.
		waterEntries, err := loadTodayEntries(ctx, store, projectID, storage.WaterDir)
		if err != nil {
			return helpers.ErrorResult("storage_error", err.Error()), nil
		}

		totalML := 0
		for _, entry := range waterEntries {
			if amt, ok := entry["amount_ml"].(float64); ok {
				totalML += int(amt)
			}
		}

		// Read profile for target.
		targetML := 2500 // default target
		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			m := meta.AsMap()
			if t, ok := m["water_target_ml"].(float64); ok && t > 0 {
				targetML = int(t)
			}
		}

		deficitML := targetML - totalML
		if deficitML < 0 {
			deficitML = 0
		}

		percentage := 0.0
		if targetML > 0 {
			percentage = float64(totalML) / float64(targetML) * 100
			if percentage > 100 {
				percentage = 100
			}
		}

		// Calculate hourly rate needed for remaining waking hours.
		nowH, nowM := nowHourMinute()
		nowMinutes := minutesSinceMidnight(nowH, nowM)

		// Default end of day at 22:00.
		endMinutes := minutesSinceMidnight(22, 0)
		if meta != nil {
			m := meta.AsMap()
			if st, ok := m["sleep_time"].(string); ok && st != "" {
				sh, sm := parseTime(st)
				if sh > 0 || sm > 0 {
					endMinutes = minutesSinceMidnight(sh, sm)
				}
			}
		}

		remainingMinutes := endMinutes - nowMinutes
		hourlyRateNeeded := 0.0
		if remainingMinutes > 0 && deficitML > 0 {
			remainingHours := float64(remainingMinutes) / 60.0
			hourlyRateNeeded = float64(deficitML) / remainingHours
		}

		status := "on_track"
		if percentage < 25 {
			status = "critical"
		} else if percentage < 50 {
			status = "behind"
		} else if percentage < 75 {
			status = "moderate"
		}

		result := map[string]any{
			"total_ml":           totalML,
			"target_ml":          targetML,
			"deficit_ml":         deficitML,
			"percentage":         fmt.Sprintf("%.1f", percentage),
			"status":             status,
			"hourly_rate_needed": fmt.Sprintf("%.0f ml/hr", hourlyRateNeeded),
			"entries_today":      len(waterEntries),
		}

		resp, err := helpers.JSONResult(result)
		if err != nil {
			return helpers.ErrorResult("internal_error", fmt.Sprintf("format result: %v", err)), nil
		}
		return resp, nil
	}
}

// ---------- nutrition_safety_check ----------

// NutritionSafetyCheckSchema returns the JSON Schema for the nutrition_safety_check tool.
func NutritionSafetyCheckSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"food_name": map[string]any{"type": "string", "description": "Name of the food to check"},
			"conditions": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Health conditions to check against (optional, reads from profile if not provided)",
			},
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
		"required": []any{"food_name"},
	})
	return s
}

// NutritionSafetyCheck returns a tool handler that checks food safety against health conditions.
func NutritionSafetyCheck(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "food_name"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		foodName := helpers.GetString(req.Arguments, "food_name")
		conditions := helpers.GetStringSlice(req.Arguments, "conditions")
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// If no conditions provided, read from profile.
		if len(conditions) == 0 {
			meta, _, _, err := store.ReadProfile(ctx, projectID)
			if err == nil && meta != nil {
				m := meta.AsMap()
				if condList, ok := m["conditions"].([]any); ok {
					for _, c := range condList {
						if cs, ok := c.(string); ok {
							conditions = append(conditions, cs)
						}
					}
				}
			}
		}

		if len(conditions) == 0 {
			result := map[string]any{
				"food_name":            foodName,
				"is_safe":              true,
				"triggered_conditions": []any{},
				"alternatives":         []any{},
				"message":              "No health conditions configured. Food is presumed safe.",
			}
			resp, err := helpers.JSONResult(result)
			if err != nil {
				return helpers.ErrorResult("internal_error", err.Error()), nil
			}
			return resp, nil
		}

		foodLower := strings.ToLower(foodName)
		var triggeredConditions []any
		alternativesSet := make(map[string]bool)

		for _, condition := range conditions {
			condLower := strings.ToLower(condition)
			blacklist, ok := conditionBlacklists[condLower]
			if !ok {
				continue
			}

			for _, banned := range blacklist {
				if containsIgnoreCase(foodLower, banned) || containsIgnoreCase(banned, foodLower) {
					triggeredConditions = append(triggeredConditions, map[string]any{
						"condition": condition,
						"trigger":   banned,
					})
					// Add alternatives for this condition.
					if alts, ok := conditionAlternatives[condLower]; ok {
						for _, alt := range alts {
							alternativesSet[alt] = true
						}
					}
				}
			}
		}

		isSafe := len(triggeredConditions) == 0

		var alternatives []any
		for alt := range alternativesSet {
			alternatives = append(alternatives, alt)
		}

		message := "Food is safe for all checked conditions."
		if !isSafe {
			message = fmt.Sprintf("WARNING: %s may trigger issues with %d condition(s).", foodName, len(triggeredConditions))
		}

		result := map[string]any{
			"food_name":            foodName,
			"is_safe":              isSafe,
			"triggered_conditions": triggeredConditions,
			"alternatives":         alternatives,
			"message":              message,
		}

		resp, err := helpers.JSONResult(result)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}
		return resp, nil
	}
}

// ---------- get_shutdown_status ----------

// GetShutdownStatusSchema returns the JSON Schema for the get_shutdown_status tool.
func GetShutdownStatusSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// GetShutdownStatus returns a tool handler that calculates shutdown status.
func GetShutdownStatus(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// Read profile for sleep config.
		sleepTimeStr := "22:00"
		shutdownBeforeMin := 60 // default 60 min before sleep

		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			m := meta.AsMap()
			if st, ok := m["sleep_time"].(string); ok && st != "" {
				sleepTimeStr = st
			}
			if sb, ok := m["shutdown_before_min"].(float64); ok && sb > 0 {
				shutdownBeforeMin = int(sb)
			}
		}

		sleepH, sleepM := parseTime(sleepTimeStr)
		sleepMinutes := minutesSinceMidnight(sleepH, sleepM)
		shutdownMinutes := sleepMinutes - shutdownBeforeMin

		nowH, nowM := nowHourMinute()
		nowMinutes := minutesSinceMidnight(nowH, nowM)

		minutesUntilShutdown := shutdownMinutes - nowMinutes
		minutesUntilSleep := sleepMinutes - nowMinutes

		isShutdownActive := nowMinutes >= shutdownMinutes
		isPastSleep := nowMinutes >= sleepMinutes

		// Check if there is an active shutdown entry today.
		shutdownEntries, _ := loadTodayEntries(ctx, store, projectID, storage.ShutdownDir)
		hasActiveShutdown := false
		for _, entry := range shutdownEntries {
			if status, ok := entry["status"].(string); ok && status == "active" {
				hasActiveShutdown = true
				break
			}
		}

		compliant := true
		if isShutdownActive && !hasActiveShutdown {
			compliant = false
		}

		status := "before_shutdown"
		if isPastSleep {
			status = "past_sleep_time"
		} else if isShutdownActive {
			status = "shutdown_active"
		}

		shutdownTime := fmt.Sprintf("%02d:%02d", shutdownMinutes/60, shutdownMinutes%60)

		result := map[string]any{
			"status":                status,
			"sleep_time":            sleepTimeStr,
			"shutdown_time":         shutdownTime,
			"minutes_until_shutdown": minutesUntilShutdown,
			"minutes_until_sleep":   minutesUntilSleep,
			"shutdown_active":       isShutdownActive,
			"has_started_routine":   hasActiveShutdown,
			"compliant":             compliant,
		}

		resp, err := helpers.JSONResult(result)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}
		return resp, nil
	}
}

// ---------- get_caffeine_window ----------

// GetCaffeineWindowSchema returns the JSON Schema for the get_caffeine_window tool.
func GetCaffeineWindowSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// GetCaffeineWindow returns a tool handler that checks if the current time is within
// the safe caffeine window based on the cortisol cycle.
func GetCaffeineWindow(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// Defaults.
		wakeTimeStr := "07:00"
		caffeineDelayMin := 90
		caffeineCutoffStr := "14:00"

		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			m := meta.AsMap()
			if wt, ok := m["wake_time"].(string); ok && wt != "" {
				wakeTimeStr = wt
			}
			if cd, ok := m["caffeine_delay_min"].(float64); ok && cd > 0 {
				caffeineDelayMin = int(cd)
			}
			if cc, ok := m["caffeine_cutoff"].(string); ok && cc != "" {
				caffeineCutoffStr = cc
			}
		}

		wakeH, wakeM := parseTime(wakeTimeStr)
		wakeMinutes := minutesSinceMidnight(wakeH, wakeM)
		safeStartMinutes := wakeMinutes + caffeineDelayMin

		cutoffH, cutoffM := parseTime(caffeineCutoffStr)
		cutoffMinutes := minutesSinceMidnight(cutoffH, cutoffM)

		nowH, nowM := nowHourMinute()
		nowMinutes := minutesSinceMidnight(nowH, nowM)

		inCortisolWindow := nowMinutes < safeStartMinutes
		pastCutoff := nowMinutes >= cutoffMinutes
		withinWindow := !inCortisolWindow && !pastCutoff

		safeTime := fmt.Sprintf("%02d:%02d", safeStartMinutes/60, safeStartMinutes%60)

		recommendation := "Safe to consume caffeine."
		if inCortisolWindow {
			minutesUntilSafe := safeStartMinutes - nowMinutes
			recommendation = fmt.Sprintf("In cortisol peak window. Wait %d minutes (until %s) for caffeine.", minutesUntilSafe, safeTime)
		} else if pastCutoff {
			recommendation = fmt.Sprintf("Past caffeine cutoff (%s). Avoid caffeine to protect sleep quality.", caffeineCutoffStr)
		}

		result := map[string]any{
			"within_window":     withinWindow,
			"in_cortisol_window": inCortisolWindow,
			"past_cutoff":       pastCutoff,
			"wake_time":         wakeTimeStr,
			"safe_time":         safeTime,
			"cutoff_time":       caffeineCutoffStr,
			"recommendation":    recommendation,
		}

		resp, err := helpers.JSONResult(result)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}
		return resp, nil
	}
}

// ---------- get_health_summary ----------

// GetHealthSummarySchema returns the JSON Schema for the get_health_summary tool.
func GetHealthSummarySchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// GetHealthSummary returns a tool handler that produces a comprehensive health summary for today.
func GetHealthSummary(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		// Water
		waterEntries, _ := loadTodayEntries(ctx, store, projectID, storage.WaterDir)
		totalWaterML := 0
		for _, e := range waterEntries {
			if amt, ok := e["amount_ml"].(float64); ok {
				totalWaterML += int(amt)
			}
		}

		// Meals
		mealEntries, _ := loadTodayEntries(ctx, store, projectID, storage.MealsDir)
		totalCalories := 0
		mealCount := len(mealEntries)
		for _, e := range mealEntries {
			if cal, ok := e["calories"].(float64); ok {
				totalCalories += int(cal)
			}
		}

		// Caffeine
		caffeineEntries, _ := loadTodayEntries(ctx, store, projectID, storage.CaffeineDir)
		totalCaffeineMG := 0
		for _, e := range caffeineEntries {
			if amt, ok := e["amount_mg"].(float64); ok {
				totalCaffeineMG += int(amt)
			}
		}

		// Pomodoros
		pomodoroEntries, _ := loadTodayEntries(ctx, store, projectID, storage.PomodoroDir)
		completedPomodoros := 0
		activePomodoros := 0
		totalFocusMin := 0
		for _, e := range pomodoroEntries {
			status, _ := e["status"].(string)
			duration, _ := e["duration_min"].(float64)
			if status == "completed" {
				completedPomodoros++
				totalFocusMin += int(duration)
			} else if status == "active" {
				activePomodoros++
			}
		}

		// Profile-based scoring
		waterTarget := 2500
		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			m := meta.AsMap()
			if t, ok := m["water_target_ml"].(float64); ok && t > 0 {
				waterTarget = int(t)
			}
		}

		hydrationScore := float64(totalWaterML) / float64(waterTarget) * 100
		if hydrationScore > 100 {
			hydrationScore = 100
		}

		focusScore := float64(completedPomodoros) / 8.0 * 100 // 8 pomodoros = 100%
		if focusScore > 100 {
			focusScore = 100
		}

		caffeineScore := 100.0
		if totalCaffeineMG > 400 { // FDA recommended max
			caffeineScore = 50.0
		} else if totalCaffeineMG > 300 {
			caffeineScore = 75.0
		}

		overallScore := (hydrationScore + focusScore + caffeineScore) / 3.0

		result := map[string]any{
			"date": todayPrefix(),
			"hydration": map[string]any{
				"total_ml":  totalWaterML,
				"target_ml": waterTarget,
				"entries":   len(waterEntries),
				"score":     fmt.Sprintf("%.0f", hydrationScore),
			},
			"nutrition": map[string]any{
				"meals":          mealCount,
				"total_calories": totalCalories,
			},
			"caffeine": map[string]any{
				"total_mg": totalCaffeineMG,
				"entries":  len(caffeineEntries),
				"score":    fmt.Sprintf("%.0f", caffeineScore),
			},
			"focus": map[string]any{
				"completed_pomodoros": completedPomodoros,
				"active_pomodoros":    activePomodoros,
				"total_focus_min":     totalFocusMin,
				"score":               fmt.Sprintf("%.0f", focusScore),
			},
			"overall_score": fmt.Sprintf("%.0f", overallScore),
		}

		resp, err := helpers.JSONResult(result)
		if err != nil {
			return helpers.ErrorResult("internal_error", err.Error()), nil
		}
		return resp, nil
	}
}

// ---------- health_reminder ----------

// HealthReminderSchema returns the JSON Schema for the health_reminder tool.
func HealthReminderSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_id": map[string]any{"type": "string", "description": "Project slug (optional, defaults to global)"},
		},
	})
	return s
}

// HealthReminder returns a tool handler that generates contextual health reminders.
func HealthReminder(store *storage.DataStorage) ToolHandler {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		projectID := storage.ResolveProject(helpers.GetString(req.Arguments, "project_id"))

		var reminders []string

		// Load profile defaults.
		wakeTimeStr := "07:00"
		sleepTimeStr := "22:00"
		caffeineDelayMin := 90
		caffeineCutoffStr := "14:00"
		waterTarget := 2500
		shutdownBeforeMin := 60

		meta, _, _, err := store.ReadProfile(ctx, projectID)
		if err == nil && meta != nil {
			m := meta.AsMap()
			if wt, ok := m["wake_time"].(string); ok && wt != "" {
				wakeTimeStr = wt
			}
			if st, ok := m["sleep_time"].(string); ok && st != "" {
				sleepTimeStr = st
			}
			if cd, ok := m["caffeine_delay_min"].(float64); ok && cd > 0 {
				caffeineDelayMin = int(cd)
			}
			if cc, ok := m["caffeine_cutoff"].(string); ok && cc != "" {
				caffeineCutoffStr = cc
			}
			if wt, ok := m["water_target_ml"].(float64); ok && wt > 0 {
				waterTarget = int(wt)
			}
			if sb, ok := m["shutdown_before_min"].(float64); ok && sb > 0 {
				shutdownBeforeMin = int(sb)
			}
		}

		nowH, nowM := nowHourMinute()
		nowMinutes := minutesSinceMidnight(nowH, nowM)

		// Check work window (between wake and sleep).
		wakeH, wakeM := parseTime(wakeTimeStr)
		wakeMinutes := minutesSinceMidnight(wakeH, wakeM)
		sleepH, sleepM := parseTime(sleepTimeStr)
		sleepMinutes := minutesSinceMidnight(sleepH, sleepM)
		shutdownMinutes := sleepMinutes - shutdownBeforeMin

		if nowMinutes < wakeMinutes {
			reminders = append(reminders, "It is before your wake time. Get some rest!")
		}

		// 1. Water deficit check.
		waterEntries, _ := loadTodayEntries(ctx, store, projectID, storage.WaterDir)
		totalWaterML := 0
		for _, e := range waterEntries {
			if amt, ok := e["amount_ml"].(float64); ok {
				totalWaterML += int(amt)
			}
		}
		deficit := waterTarget - totalWaterML
		if deficit > 500 {
			reminders = append(reminders,
				fmt.Sprintf("Hydration alert: You are %d ml behind your water target (%d/%d ml). Drink a glass of water now.", deficit, totalWaterML, waterTarget))
		}

		// 2. Pomodoro break check: if no pomodoro completed in the last 90 minutes.
		pomodoroEntries, _ := loadTodayEntries(ctx, store, projectID, storage.PomodoroDir)
		lastPomodoroMinutes := 0
		for _, e := range pomodoroEntries {
			if ca, ok := e["created_at"].(string); ok && len(ca) >= 16 {
				// Parse HH:MM from ISO timestamp.
				timeStr := ca[11:16]
				ph, pm := parseTime(timeStr)
				pMin := minutesSinceMidnight(ph, pm)
				if pMin > lastPomodoroMinutes {
					lastPomodoroMinutes = pMin
				}
			}
		}
		if lastPomodoroMinutes == 0 || (nowMinutes-lastPomodoroMinutes) > 90 {
			if nowMinutes > wakeMinutes && nowMinutes < shutdownMinutes {
				reminders = append(reminders,
					"Focus break: No pomodoro session in the last 90 minutes. Consider a standing break and starting a new pomodoro.")
			}
		}

		// 3. Shutdown window check.
		if nowMinutes >= shutdownMinutes && nowMinutes < sleepMinutes {
			reminders = append(reminders,
				fmt.Sprintf("Shutdown reminder: You are within %d minutes of sleep time. Start your shutdown routine (GERD protocol, screen reduction, sleep prep).", sleepMinutes-nowMinutes))
		}

		// 4. Caffeine in cortisol window check.
		caffeineEntries, _ := loadTodayEntries(ctx, store, projectID, storage.CaffeineDir)
		safeStartMinutes := wakeMinutes + caffeineDelayMin
		cutoffH, cutoffM := parseTime(caffeineCutoffStr)
		cutoffMinutes := minutesSinceMidnight(cutoffH, cutoffM)

		if nowMinutes < safeStartMinutes {
			reminders = append(reminders,
				fmt.Sprintf("Cortisol window: Your cortisol is naturally high right now. Wait until %02d:%02d for caffeine.", safeStartMinutes/60, safeStartMinutes%60))
		}
		if nowMinutes >= cutoffMinutes {
			// Check if any caffeine logged after cutoff.
			for _, e := range caffeineEntries {
				if ca, ok := e["created_at"].(string); ok && len(ca) >= 16 {
					timeStr := ca[11:16]
					ch, cm := parseTime(timeStr)
					cMin := minutesSinceMidnight(ch, cm)
					if cMin >= cutoffMinutes {
						reminders = append(reminders,
							fmt.Sprintf("Caffeine warning: Caffeine consumed after cutoff time (%s). This may affect sleep quality.", caffeineCutoffStr))
						break
					}
				}
			}
		}

		if len(reminders) == 0 {
			reminders = append(reminders, "All clear! No health reminders at this time. Keep up the good work.")
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Health Reminders (%02d:%02d UTC)\n\n", nowH, nowM)
		for i, r := range reminders {
			fmt.Fprintf(&b, "%d. %s\n", i+1, r)
		}

		return helpers.TextResult(b.String()), nil
	}
}
