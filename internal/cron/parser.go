// Package cron parses 5-field cron expressions into launchd-compatible
// calendar intervals. It supports lists, ranges, steps, and the common
// @-shortcuts (@reboot, @daily, @hourly, @weekly, @monthly, @yearly).
//
// launchd's StartCalendarInterval only accepts single values per key, so a
// cron expression that covers multiple minutes/hours/etc. is expanded into
// the cartesian product of its field values.
package cron

import (
	"fmt"
	"strconv"
	"strings"
)

// Schedule is the launchd-friendly representation of a cron expression.
// If RunAtLoad is true (from @reboot), Intervals will be empty.
type Schedule struct {
	RunAtLoad bool
	Intervals []Interval
}

// Interval mirrors a single StartCalendarInterval dict entry. A nil pointer
// means "any value" — i.e. the key is omitted from the emitted plist.
type Interval struct {
	Minute  *int
	Hour    *int
	Day     *int
	Month   *int
	Weekday *int
}

type fieldSpec struct {
	name     string
	min, max int
}

var (
	minuteField  = fieldSpec{"minute", 0, 59}
	hourField    = fieldSpec{"hour", 0, 23}
	dayField     = fieldSpec{"day", 1, 31}
	monthField   = fieldSpec{"month", 1, 12}
	weekdayField = fieldSpec{"weekday", 0, 7} // 0 and 7 both mean Sunday
)

// Parse converts a cron expression into a launchd Schedule.
func Parse(expr string) (*Schedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}

	if strings.HasPrefix(expr, "@") {
		return parseShortcut(expr)
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	minutes, err := parseField(fields[0], minuteField)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hours, err := parseField(fields[1], hourField)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	days, err := parseField(fields[2], dayField)
	if err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	months, err := parseField(fields[3], monthField)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	weekdays, err := parseField(fields[4], weekdayField)
	if err != nil {
		return nil, fmt.Errorf("weekday: %w", err)
	}

	// Normalize weekday: launchd uses 0=Sun..6=Sat. Cron also allows 7=Sun.
	weekdays = normalizeWeekdays(weekdays)

	// Build cartesian product. A wildcard field is signalled by nil slice
	// so the corresponding Interval pointer stays nil (omitted from plist).
	return &Schedule{
		Intervals: cartesian(minutes, hours, days, months, weekdays),
	}, nil
}

func parseShortcut(expr string) (*Schedule, error) {
	switch expr {
	case "@reboot":
		return &Schedule{RunAtLoad: true}, nil
	case "@hourly":
		return Parse("0 * * * *")
	case "@daily", "@midnight":
		return Parse("0 0 * * *")
	case "@weekly":
		return Parse("0 0 * * 0")
	case "@monthly":
		return Parse("0 0 1 * *")
	case "@yearly", "@annually":
		return Parse("0 0 1 1 *")
	default:
		return nil, fmt.Errorf("unknown shortcut %q", expr)
	}
}

// parseField returns the concrete values a cron field expands to.
// A pure wildcard (`*` or `*/1`) returns nil to indicate "omit from plist".
func parseField(raw string, spec fieldSpec) ([]int, error) {
	// Pure wildcard: omit the key entirely so launchd fires for every value.
	if raw == "*" {
		return nil, nil
	}

	// Lists are comma-separated; each element may itself be a range or step.
	var out []int
	seen := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		values, err := parsePart(part, spec)
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out, nil
}

func parsePart(part string, spec fieldSpec) ([]int, error) {
	step := 1
	if idx := strings.Index(part, "/"); idx >= 0 {
		s, err := strconv.Atoi(part[idx+1:])
		if err != nil || s <= 0 {
			return nil, fmt.Errorf("invalid step %q", part[idx+1:])
		}
		step = s
		part = part[:idx]
	}

	var lo, hi int
	switch {
	case part == "*":
		lo, hi = spec.min, spec.max
	case strings.Contains(part, "-"):
		bounds := strings.SplitN(part, "-", 2)
		a, err := strconv.Atoi(bounds[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range start %q", bounds[0])
		}
		b, err := strconv.Atoi(bounds[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range end %q", bounds[1])
		}
		lo, hi = a, b
	default:
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", part)
		}
		lo, hi = n, n
	}

	if lo < spec.min || hi > spec.max || lo > hi {
		return nil, fmt.Errorf("%s out of range [%d-%d]: %d-%d",
			spec.name, spec.min, spec.max, lo, hi)
	}

	var out []int
	for v := lo; v <= hi; v += step {
		out = append(out, v)
	}
	return out, nil
}

func normalizeWeekdays(weekdays []int) []int {
	if weekdays == nil {
		return nil
	}
	seen := make(map[int]bool)
	var out []int
	for _, w := range weekdays {
		if w == 7 {
			w = 0
		}
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// cartesian expands per-field value slices into concrete Interval entries.
// A nil input slice means "any" — the corresponding Interval pointer stays nil.
func cartesian(minutes, hours, days, months, weekdays []int) []Interval {
	// Use a sentinel slice of length 1 with a nil-marker so the loops still run
	// once per wildcard field without producing concrete values.
	m := orAny(minutes)
	h := orAny(hours)
	d := orAny(days)
	mo := orAny(months)
	w := orAny(weekdays)

	var out []Interval
	for _, mv := range m {
		for _, hv := range h {
			for _, dv := range d {
				for _, mov := range mo {
					for _, wv := range w {
						out = append(out, Interval{
							Minute:  mv,
							Hour:    hv,
							Day:     dv,
							Month:   mov,
							Weekday: wv,
						})
					}
				}
			}
		}
	}
	return out
}

// orAny converts a slice of ints into a slice of *int, or a single-element
// nil slice when the input is nil (wildcard field).
func orAny(vs []int) []*int {
	if vs == nil {
		return []*int{nil}
	}
	out := make([]*int, len(vs))
	for i, v := range vs {
		v := v
		out[i] = &v
	}
	return out
}
