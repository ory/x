package pointerx

// String returns the input value's pointer.
func String(s string) *string {
	return &s
}

// Int returns the input value's pointer.
func Int(s int) *int {
	return &s
}

// Int32 returns the input value's pointer.
func Int32(s int32) *int32 {
	return &s
}

// Int64 returns the input value's pointer.
func Int64(s int64) *int64 {
	return &s
}

// Float32 returns the input value's pointer.
func Float32(s float32) *float32 {
	return &s
}

// Float64 returns the input value's pointer.
func Float64(s float64) *float64 {
	return &s
}
