package controller

// pointerTo is a generic utility for creating pointers to values in tests.
func pointerTo[T any](val T) *T {
	return &val
}
