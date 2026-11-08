package plays

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// cronSpec is a parsed five-field cron expression: each field the set of values it allows.
type cronSpec struct {
	minute, hour, dom, month, dow uint64
	// domAny and dowAny are true for a "*" field: standard cron ORs the two day fields only when both are restricted.
	domAny, dowAny bool
}

type cronField struct {
	name     string
	min, max int
	names    []string
}

var cronFields = [5]cronField{
	{name: "minute", min: 0, max: 59},
	{name: "hour", min: 0, max: 23},
	{name: "day of month", min: 1, max: 31},
	{name: "month", min: 1, max: 12, names: []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}},
	{name: "day of week", min: 0, max: 7, names: []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}},
}

// minScheduleGap is the shortest a schedule may leave between two sweeps.
const minScheduleGap = time.Hour

// parseCron reads "minute hour day-of-month month day-of-week", each field *, a value, a range, a step, or a list.
func parseCron(expr string) (cronSpec, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return cronSpec{}, fmt.Errorf("%w: a schedule has five fields, minute hour day-of-month month day-of-week, such as 0 23 * * *", apperrs.ErrInvalid)
	}
	var sets [5]uint64
	for i, part := range parts {
		set, err := cronFields[i].parse(part)
		if err != nil {
			return cronSpec{}, err
		}
		sets[i] = set
	}
	if sets[4]&(1<<7) != 0 {
		sets[4] = sets[4]&^(1<<7) | 1
	}
	spec := cronSpec{minute: sets[0], hour: sets[1], dom: sets[2], month: sets[3], dow: sets[4], domAny: parts[2] == "*", dowAny: parts[4] == "*"}
	first := spec.next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if first.IsZero() {
		return cronSpec{}, fmt.Errorf("%w: schedule %q never fires", apperrs.ErrInvalid, expr)
	}
	return spec, spec.checkGap(expr, first)
}

// checkGap refuses a schedule that fires more than once an hour, over a year of its sweeps.
func (c cronSpec) checkGap(expr string, from time.Time) error {
	for range 400 {
		next := c.next(from)
		if next.IsZero() {
			return nil
		}
		if next.Sub(from) < minScheduleGap {
			return fmt.Errorf("%w: schedule %q fires more than once an hour", apperrs.ErrInvalid, expr)
		}
		from = next
	}
	return nil
}

func (f cronField) parse(part string) (uint64, error) {
	var set uint64
	for item := range strings.SplitSeq(part, ",") {
		bits, err := f.parseItem(item)
		if err != nil {
			return 0, err
		}
		set |= bits
	}
	return set, nil
}

func (f cronField) parseItem(item string) (uint64, error) {
	span, step := item, 1
	if before, after, ok := strings.Cut(item, "/"); ok {
		n, err := strconv.Atoi(after)
		if err != nil || n < 1 {
			return 0, fmt.Errorf("%w: %s step %q must be a whole number above 0", apperrs.ErrInvalid, f.name, after)
		}
		span, step = before, n
	}
	lo, hi, err := f.parseSpan(span, step > 1)
	if err != nil {
		return 0, err
	}
	var bits uint64
	for v := lo; v <= hi; v += step {
		bits |= 1 << v
	}
	return bits, nil
}

// parseSpan reads *, a value, or a range; a stepped single value runs to the field's end, as "5/15" does in cron.
func (f cronField) parseSpan(span string, stepped bool) (int, int, error) {
	if span == "*" {
		return f.min, f.max, nil
	}
	if a, b, ok := strings.Cut(span, "-"); ok {
		lo, err := f.value(a)
		if err != nil {
			return 0, 0, err
		}
		hi, err := f.value(b)
		if err != nil {
			return 0, 0, err
		}
		if lo > hi {
			return 0, 0, fmt.Errorf("%w: %s range %q runs backwards", apperrs.ErrInvalid, f.name, span)
		}
		return lo, hi, nil
	}
	v, err := f.value(span)
	if err != nil {
		return 0, 0, err
	}
	if stepped {
		return v, f.max, nil
	}
	return v, v, nil
}

func (f cronField) value(s string) (int, error) {
	for i, name := range f.names {
		if strings.EqualFold(s, name) {
			return i + f.min, nil
		}
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < f.min || v > f.max {
		return 0, fmt.Errorf("%w: %s %q must be %d to %d", apperrs.ErrInvalid, f.name, s, f.min, f.max)
	}
	return v, nil
}

func (c cronSpec) dayMatches(t time.Time) bool {
	dom, dow := c.dom&(1<<t.Day()) != 0, c.dow&(1<<int(t.Weekday())) != 0
	if c.domAny || c.dowAny {
		return dom && dow
	}
	return dom || dow
}

// next is the first time after from that the spec allows, read in from's location; zero when none in five years.
func (c cronSpec) next(from time.Time) time.Time {
	loc := from.Location()
	t := from.Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(5, 0, 0)
	for t.Before(limit) {
		if c.month&(1<<int(t.Month())) == 0 {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !c.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			continue
		}
		if c.hour&(1<<t.Hour()) == 0 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, loc)
			continue
		}
		if c.minute&(1<<t.Minute()) == 0 {
			t = t.Add(time.Minute)
			continue
		}
		return t
	}
	return time.Time{}
}
