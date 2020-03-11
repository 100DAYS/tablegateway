package tablegateway

import (
	"database/sql"
	"time"
)

func NullString(s string) sql.NullString {
	return sql.NullString{Valid: true, String: s}
}

func NullInt64(s int64) sql.NullInt64 {
	return sql.NullInt64{Valid: true, Int64: s}
}

func NullInt32(s int32) sql.NullInt32 {
	return sql.NullInt32{Valid: true, Int32: s}
}

func NullFloat64(s float64) sql.NullFloat64 {
	return sql.NullFloat64{Valid: true, Float64: s}
}

func NullBool(s bool) sql.NullBool {
	return sql.NullBool{Valid: true, Bool: s}
}

func NullTime(s time.Time) sql.NullTime {
	return sql.NullTime{Valid: true, Time: s}
}
