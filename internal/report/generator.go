package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"dockaudit/internal/planner"
)

type ReportData struct {
	Timestamp          time.Time        `json:"timestamp"`
	TotalRecoveryBytes int64            `json:"total_recovery_bytes"`
	SafeActions        []planner.Action `json:"safe_actions"`
	RiskyActions       []planner.Action `json:"risky_actions"`
}

func GenerateJSON(plan *planner.Plan, outputFile string) error {
	data := buildReportData(plan)
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, bytes, 0644)
}

func GenerateMarkdown(plan *planner.Plan, outputFile string) error {
	data := buildReportData(plan)
	
	md := fmt.Sprintf("# Dockaudit Report\n\n**Generated:** %s\n\n", data.Timestamp.Format(time.RFC1123))
	md += fmt.Sprintf("## Recovery Potential: %.2f GB\n\n", float64(data.TotalRecoveryBytes)/(1024*1024*1024))
	
	md += "### Safe to Delete\n\n"
	for _, a := range data.SafeActions {
		md += fmt.Sprintf("- **[%s]** `%s`\n  - Reason: %s\n", a.ResourceType, a.Name, a.Reason)
	}

	md += "\n### Requires Review / Risky\n\n"
	for _, a := range data.RiskyActions {
		md += fmt.Sprintf("- **[%s]** `%s`\n  - Risk: %s\n  - Reason: %s\n", a.ResourceType, a.Name, a.Risk, a.Reason)
	}

	return os.WriteFile(outputFile, []byte(md), 0644)
}

func GenerateHTML(plan *planner.Plan, outputFile string) error {
	data := buildReportData(plan)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Dockaudit Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; padding: 2rem; max-width: 900px; margin: 0 auto; color: #333; }
        h1 { color: #2c3e50; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        .recovery { background: #e8f5e9; color: #2e7d32; padding: 1rem; border-radius: 8px; font-size: 1.2rem; font-weight: bold; margin-bottom: 2rem; }
        .section { margin-top: 2rem; }
        .item { border: 1px solid #ddd; padding: 1rem; margin-bottom: 1rem; border-radius: 6px; }
        .item-header { display: flex; justify-content: space-between; font-weight: bold; margin-bottom: 0.5rem; }
        .badge { padding: 0.2rem 0.5rem; border-radius: 4px; font-size: 0.8rem; }
        .badge-safe { background: #e8f5e9; color: #2e7d32; }
        .badge-risky { background: #ffebee; color: #c62828; }
        .reason { color: #666; font-size: 0.9rem; }
    </style>
</head>
<body>
    <h1>Dockaudit Report</h1>
    <p>Generated: %s</p>
    <div class="recovery">Recovery Potential: %.2f GB</div>
`, data.Timestamp.Format(time.RFC1123), float64(data.TotalRecoveryBytes)/(1024*1024*1024))

	html += `<div class="section"><h2>Safe to Delete</h2>`
	for _, a := range data.SafeActions {
		html += fmt.Sprintf(`
		<div class="item">
			<div class="item-header">
				<span>%s</span>
				<span class="badge badge-safe">%s</span>
			</div>
			<div class="reason">%s</div>
		</div>`, a.Name, a.ResourceType, a.Reason)
	}
	html += `</div>`

	html += `<div class="section"><h2>Requires Review / Risky</h2>`
	for _, a := range data.RiskyActions {
		html += fmt.Sprintf(`
		<div class="item">
			<div class="item-header">
				<span>%s</span>
				<span class="badge badge-risky">%s</span>
			</div>
			<div class="reason">%s: %s</div>
		</div>`, a.Name, a.ResourceType, a.Risk, a.Reason)
	}
	html += `</div></body></html>`

	return os.WriteFile(outputFile, []byte(html), 0644)
}

func buildReportData(plan *planner.Plan) ReportData {
	data := ReportData{
		Timestamp:          time.Now(),
		TotalRecoveryBytes: plan.TotalRecoveryBytes,
		SafeActions:        make([]planner.Action, 0),
		RiskyActions:       make([]planner.Action, 0),
	}

	for _, a := range plan.Actions {
		if a.Risk == planner.RiskSafe || a.Risk == planner.RiskLikelySafe {
			data.SafeActions = append(data.SafeActions, a)
		} else {
			data.RiskyActions = append(data.RiskyActions, a)
		}
	}

	return data
}
