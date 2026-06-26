package model

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const localTimeFormat = "2006-01-02 15:04:05"

// LocalTime 本地时间，JSON 输出 "2006-01-02 15:04:05"
type LocalTime struct {
	T time.Time
}

var _ driver.Valuer = LocalTime{}
var _ sql.Scanner = (*LocalTime)(nil)

func Now() LocalTime {
	return LocalTime{T: time.Now()}
}

func (lt LocalTime) Ptr() *LocalTime {
	return &lt
}

func (lt LocalTime) IsZero() bool {
	return lt.T.IsZero()
}

func (lt LocalTime) Time() time.Time {
	return lt.T
}

func (lt LocalTime) MarshalJSON() ([]byte, error) {
	if lt.T.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + lt.T.Format(localTimeFormat) + `"`), nil
}

func (lt *LocalTime) UnmarshalJSON(data []byte) error {
	s := string(data)
	s = strings.Trim(s, `"`)
	if s == "" || s == "null" {
		return nil
	}
	formats := []string{
		localTimeFormat,
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			lt.T = t
			return nil
		}
	}
	return fmt.Errorf("无法解析时间: %s", s)
}

func (lt *LocalTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		lt.T = v
		return nil
	case []byte:
		t, err := time.Parse(localTimeFormat, string(v))
		if err != nil {
			return err
		}
		lt.T = t
		return nil
	case string:
		t, err := time.Parse(localTimeFormat, v)
		if err != nil {
			return err
		}
		lt.T = t
		return nil
	}
	return fmt.Errorf("无法转换 %T 为 LocalTime", value)
}

func (lt LocalTime) Value() (driver.Value, error) {
	if lt.T.IsZero() {
		return nil, nil
	}
	return lt.T, nil
}
