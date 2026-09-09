package sqlitehelpers

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		s = StringSlice{}
	}

	encoded, err := json.Marshal([]string(s))
	if err != nil {
		return nil, fmt.Errorf("marshalling string slice: %w", err)
	}

	return string(encoded), nil
}

func (s *StringSlice) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case nil:
		*s = StringSlice{}
		return nil
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return fmt.Errorf("cannot scan %T into StringSlice", src)
	}

	if len(raw) == 0 {
		*s = StringSlice{}
		return nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return fmt.Errorf("unmarshalling string slice: %w", err)
	}
	if values == nil {
		values = []string{}
	}

	*s = values
	return nil
}
