package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// Date wraps time.Time but accepts both "2006-01-02" (HTML date inputs) and
// full RFC3339 timestamps on JSON input — Go's default time.Time only accepts
// RFC3339, which breaks date-only fields coming from the UI.
type Date struct{ time.Time }

func NewDate(t time.Time) Date { return Date{Time: t} }

var dateLayouts = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("tarix formatı yanlışdır: %s", s)
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Time.Format(time.RFC3339) + `"`), nil
}

// SQL / GORM integration.
func (d Date) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time, nil
}

func (d *Date) Scan(v interface{}) error {
	if v == nil {
		d.Time = time.Time{}
		return nil
	}
	if t, ok := v.(time.Time); ok {
		d.Time = t
		return nil
	}
	return fmt.Errorf("Date.Scan: dəstəklənməyən tip %T", v)
}

func (Date) GormDataType() string { return "timestamptz" }
