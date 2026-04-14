// Package plist renders a launchd property list from a Job description.
//
// We hand-write the XML rather than depend on a third-party plist library —
// the launchd subset we care about is narrow, the output is stable, and
// zero-dep keeps `go install` and vendoring trivial.
package plist

import (
	"fmt"
	"io"
	"strings"

	"github.com/VandyTheCoder/launchd-gen/internal/cron"
)

// Job is everything launchd needs to know about a scheduled task.
type Job struct {
	Label            string
	ProgramArguments []string
	WorkingDirectory string
	StandardOutPath  string
	StandardErrPath  string
	EnvironmentVars  map[string]string
	Schedule         *cron.Schedule
}

const header = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`

const footer = `</dict>
</plist>
`

// Write emits a launchd-compatible plist for the given Job to w.
func Write(w io.Writer, j Job) error {
	if j.Label == "" {
		return fmt.Errorf("Label is required")
	}
	if len(j.ProgramArguments) == 0 {
		return fmt.Errorf("ProgramArguments is required")
	}

	var b strings.Builder
	b.WriteString(header)

	writeStringKey(&b, "Label", j.Label)
	writeStringArrayKey(&b, "ProgramArguments", j.ProgramArguments)

	if j.WorkingDirectory != "" {
		writeStringKey(&b, "WorkingDirectory", j.WorkingDirectory)
	}
	if j.StandardOutPath != "" {
		writeStringKey(&b, "StandardOutPath", j.StandardOutPath)
	}
	if j.StandardErrPath != "" {
		writeStringKey(&b, "StandardErrPath", j.StandardErrPath)
	}
	if len(j.EnvironmentVars) > 0 {
		writeEnvKey(&b, j.EnvironmentVars)
	}

	if j.Schedule != nil {
		if j.Schedule.RunAtLoad {
			writeBoolKey(&b, "RunAtLoad", true)
		}
		if len(j.Schedule.Intervals) > 0 {
			writeIntervalsKey(&b, j.Schedule.Intervals)
		}
	}

	b.WriteString(footer)
	_, err := io.WriteString(w, b.String())
	return err
}

func writeStringKey(b *strings.Builder, key, val string) {
	fmt.Fprintf(b, "    <key>%s</key>\n    <string>%s</string>\n", key, escape(val))
}

func writeBoolKey(b *strings.Builder, key string, val bool) {
	tag := "false"
	if val {
		tag = "true"
	}
	fmt.Fprintf(b, "    <key>%s</key>\n    <%s/>\n", key, tag)
}

func writeStringArrayKey(b *strings.Builder, key string, vals []string) {
	fmt.Fprintf(b, "    <key>%s</key>\n    <array>\n", key)
	for _, v := range vals {
		fmt.Fprintf(b, "        <string>%s</string>\n", escape(v))
	}
	b.WriteString("    </array>\n")
}

func writeEnvKey(b *strings.Builder, env map[string]string) {
	b.WriteString("    <key>EnvironmentVariables</key>\n    <dict>\n")
	// Sort for deterministic output.
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "        <key>%s</key>\n        <string>%s</string>\n", escape(k), escape(env[k]))
	}
	b.WriteString("    </dict>\n")
}

func writeIntervalsKey(b *strings.Builder, intervals []cron.Interval) {
	// launchd accepts either a single dict or an array of dicts.
	// We always emit an array for consistency.
	b.WriteString("    <key>StartCalendarInterval</key>\n    <array>\n")
	for _, iv := range intervals {
		b.WriteString("        <dict>\n")
		writeOptInt(b, "Minute", iv.Minute)
		writeOptInt(b, "Hour", iv.Hour)
		writeOptInt(b, "Day", iv.Day)
		writeOptInt(b, "Month", iv.Month)
		writeOptInt(b, "Weekday", iv.Weekday)
		b.WriteString("        </dict>\n")
	}
	b.WriteString("    </array>\n")
}

func writeOptInt(b *strings.Builder, key string, v *int) {
	if v == nil {
		return
	}
	fmt.Fprintf(b, "            <key>%s</key>\n            <integer>%d</integer>\n", key, *v)
}

// escape handles the five characters that must be escaped in XML text.
func escape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// sortStrings is a tiny dep-free insertion sort for small env maps.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
