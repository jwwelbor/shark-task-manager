package sprint

import (
	"fmt"
	"time"
)

// flexTime is a sql.Scanner that accepts time.Time, int64 (Unix nanoseconds),
// or a string in any of the common datetime layouts returned by SQLite backends.
//
// Local modernc.org/sqlite returns time.Time directly; Turso/libSQL returns
// strings. Both are handled here so sprint scanning is backend-agnostic.
type flexTime struct {
	t *time.Time
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func (f flexTime) Scan(value interface{}) error {
	if value == nil {
		*f.t = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*f.t = v.UTC()
	case int64:
		// modernc.org/sqlite may return Unix nanoseconds for DATETIME columns
		*f.t = time.Unix(v/1e9, v%1e9).UTC()
	case string:
		for _, layout := range timeLayouts {
			if t, err := time.Parse(layout, v); err == nil {
				*f.t = t.UTC()
				return nil
			}
		}
		return fmt.Errorf("sprint: cannot parse time value %q", v)
	default:
		return fmt.Errorf("sprint: unsupported time value type %T", value)
	}
	return nil
}

// flexNullTime is a sql.Scanner for *time.Time that handles NULL, time.Time,
// int64, and string inputs — same layout set as flexTime.
type flexNullTime struct {
	t **time.Time
}

func (f flexNullTime) Scan(value interface{}) error {
	if value == nil {
		*f.t = nil
		return nil
	}
	var parsed time.Time
	if err := (flexTime{&parsed}).Scan(value); err != nil {
		return err
	}
	*f.t = &parsed
	return nil
}
