// Package dashboard embeds the telemetry dashboard HTML at compile time.
// The source lives at dashboard/dashboard.html and was imported from
// transpara-ai/summary via git subtree; edit it in place.
package dashboard

import _ "embed"

//go:embed dashboard.html
var HTML []byte
