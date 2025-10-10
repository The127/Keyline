// Go
package helpers

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func PqInterval(duration time.Duration) string {
	return fmt.Sprintf("%d %s", int(duration.Seconds()), "seconds")
}

// PqIntervalPtr returns a pointer to the interval string or nil if d is nil.
func PqIntervalPtr(d *time.Duration) *string {
	if d == nil {
		return nil
	}
	s := PqInterval(*d)
	return &s
}

// DurationFromPqIntervalPtr parses a pointer to a textual Postgres interval.
// Returns nil if s is nil.
func DurationFromPqIntervalPtr(s *string) *time.Duration {
	if s == nil {
		return nil
	}
	d := DurationFromPqInterval(*s)
	return &d
}

// NullDuration implements sql.Scanner and driver.Valuer to handle nullable interval columns.
type NullDuration struct {
	Duration time.Duration
	Valid    bool
}

// Value converts the duration into a Postgres-compatible textual interval or NULL.
func (nd NullDuration) Value() (driver.Value, error) {
	if !nd.Valid {
		return nil, nil
	}
	return PqInterval(nd.Duration), nil
}

// Scan reads a database value into the NullDuration.
// Accepts textual interval (string/[]byte), numeric seconds, or nil.
func (nd *NullDuration) Scan(src any) error {
	if src == nil {
		nd.Duration = 0
		nd.Valid = false
		return nil
	}

	switch v := src.(type) {
	case string:
		nd.Duration = DurationFromPqInterval(v)
		nd.Valid = true
		return nil
	case []byte:
		nd.Duration = DurationFromPqInterval(string(v))
		nd.Valid = true
		return nil
	case time.Duration:
		nd.Duration = v
		nd.Valid = true
		return nil
	case int64:
		// Interpret as whole seconds
		nd.Duration = time.Duration(v) * time.Second
		nd.Valid = true
		return nil
	case float64:
		// Interpret as seconds (possibly fractional)
		nd.Duration = time.Duration(v * float64(time.Second))
		nd.Valid = true
		return nil
	default:
		// Fallback to string formatting
		s := fmt.Sprintf("%v", v)
		nd.Duration = DurationFromPqInterval(s)
		nd.Valid = true
		return nil
	}
}

func DurationFromPqInterval(interval string) time.Duration {
	s := strings.TrimSpace(interval)
	if s == "" {
		return 0
	}

	sign := 1
	// Trailing "ago" indicates negative in PostgreSQL textual intervals
	if strings.HasSuffix(s, "ago") {
		sign = -1
		s = strings.TrimSpace(strings.TrimSuffix(s, "ago"))
	}
	// Leading "-" also indicates negative
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimSpace(strings.TrimPrefix(s, "-"))
	}

	total := time.Duration(0)
	tokens := strings.Fields(s)

	for i := 0; i < len(tokens); {
		tok := strings.TrimSpace(tokens[i])

		// Time block like HH:MM[:SS[.frac]]
		if strings.Contains(tok, ":") {
			total += parseHMSToken(tok)
			i++
			continue
		}

		// Attempt "number unit" pairs, e.g., "10 seconds", "2 days", etc.
		val, err := strconv.ParseFloat(strings.TrimSuffix(tok, ","), 64)
		if err != nil {
			i++
			continue
		}

		if i+1 < len(tokens) {
			unit := strings.ToLower(strings.Trim(strings.TrimSuffix(tokens[i+1], ","), " "))
			switch unit {
			case "second", "seconds", "sec", "secs":
				total += time.Duration(val * float64(time.Second))
			case "minute", "minutes", "min", "mins":
				total += time.Duration(val * float64(time.Minute))
			case "hour", "hours":
				total += time.Duration(val * float64(time.Hour))
			case "day", "days":
				total += time.Duration(val * 24 * float64(time.Hour))
			case "mon", "mons", "month", "months":
				// Approximate: 30 days per month
				total += time.Duration(val * 30 * 24 * float64(time.Hour))
			case "year", "years":
				// Approximate: 365 days per year
				total += time.Duration(val * 365 * 24 * float64(time.Hour))
			default:
				// Unknown unit: ignore
			}
			i += 2
			continue
		}

		i++
	}

	// Fallback: if the entire string is a plain number, treat as seconds.
	if total == 0 {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			total = time.Duration(v * float64(time.Second))
		}
	}

	return time.Duration(sign) * total
}

func parseHMSToken(tok string) time.Duration {
	t := strings.TrimSpace(tok)
	neg := false
	if strings.HasPrefix(t, "-") {
		neg = true
		t = strings.TrimPrefix(t, "-")
	}

	parts := strings.Split(t, ":")
	var h, m, sec float64

	switch len(parts) {
	case 3:
		h, _ = strconv.ParseFloat(parts[0], 64)
		m, _ = strconv.ParseFloat(parts[1], 64)
		sec, _ = strconv.ParseFloat(parts[2], 64)
	case 2:
		// Interpret as MM:SS
		m, _ = strconv.ParseFloat(parts[0], 64)
		sec, _ = strconv.ParseFloat(parts[1], 64)
	case 1:
		// Just seconds (possibly fractional)
		sec, _ = strconv.ParseFloat(parts[0], 64)
	default:
		return 0
	}

	d := time.Duration(h*float64(time.Hour) + m*float64(time.Minute) + sec*float64(time.Second))
	if neg {
		return -d
	}
	return d
}
