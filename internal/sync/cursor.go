package sync

// Cursor tracks how many snapshot rows a consumer has applied.
type Cursor struct {
	pos int
}

// NewCursor creates a cursor.
func NewCursor() *Cursor {
	return &Cursor{}
}

// Advance moves the cursor forward.
func (c *Cursor) Advance(n int) {
	c.pos += n
}

// Position returns the current position.
func (c *Cursor) Position() int {
	return c.pos
}

// Reset rewinds the cursor.
func (c *Cursor) Reset() {
	c.pos = 0
}
