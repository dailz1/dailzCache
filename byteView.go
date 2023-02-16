package main

// A ByteView holds an immutable view of bytes.
// Internally it wraps either a []byte or a string,
// but that detail is invisible to callers.
//
// A ByteView is meant to be used as a value type, not
// a pointer (like a time.Time).
type ByteView struct {
	// // If data is non-nil, data is used, else str is used.
	data []byte
	str  string
}

// Len returns the view's length.
func (v ByteView) Len() int {
	if v.data != nil {
		return len(v.data)
	}
	return len(v.str)
}

// ByteSlice returns a copy of the data as a byte slice.
func (v ByteView) ByteSlice() []byte {
	if v.data != nil {
		return cloneBytes(v.data)
	}
	return []byte(v.str)
}

// String returns the data as a string, making a copy if necessary.
func (v ByteView) String() string {
	if v.data != nil {
		return string(v.data)
	}
	return v.str
}

func cloneBytes(data []byte) []byte {
	c := make([]byte, len(data))
	copy(c, data)
	return c
}
