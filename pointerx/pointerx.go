package pointerx

// String returns the input value's pointer.
func String(s string) *string {
	return &s
}

// StringR is the reverse to String.
func StringR(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Int returns the input value's pointer.
func Int(s int) *int {
	return &s
}

// IntR is the reverse to Int.
func IntR(s *int) int {
	if s == nil {
		return int(0)
	}
	return *s
}

// Int32 returns the input value's pointer.
func Int32(s int32) *int32 {
	return &s
}

// Int32R is the reverse to Int32.
func Int32R(s *int32) int32 {
	if s == nil {
		return int32(0)
	}
	return *s
}

// Int64 returns the input value's pointer.
func Int64(s int64) *int64 {
	return &s
}

// Int64R is the reverse to Int64.
func Int64R(s *int64) int64 {
	if s == nil {
		return int64(0)
	}
	return *s
}

// Float32 returns the input value's pointer.
func Float32(s float32) *float32 {
	return &s
}

// Float32R is the reverse to Float32.
func Float32R(s *float32) float32 {
	if s == nil {
		return float32(0)
	}
	return *s
}

// Float64 returns the input value's pointer.
func Float64(s float64) *float64 {
	return &s
}

// Float64R is the reverse to Float64.
func Float64R(s *float64) float64 {
	if s == nil {
		return float64(0)
	}
	return *s
}

// Bool returns the input value's pointer.
func Bool(s bool) *bool {
	return &s
}

// BoolR is the reverse to Bool.
func BoolR(s *bool) bool {
	if s == nil {
		return false
	}
	return *s
}
