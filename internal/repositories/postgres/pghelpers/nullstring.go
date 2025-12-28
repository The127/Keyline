package pghelpers

import "database/sql"

func WrapStringPointer(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{
			String: "",
			Valid:  false,
		}
	}

	return sql.NullString{
		String: *s,
		Valid:  true,
	}
}

func UnwrapNullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}

	value := s.String
	return &value
}
