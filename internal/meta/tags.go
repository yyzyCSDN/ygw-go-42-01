package meta

// TagSet is a set of instance tags.
type TagSet struct {
	tags map[string]bool
}

// NewTagSet creates an empty tag set.
func NewTagSet() *TagSet {
	return &TagSet{tags: make(map[string]bool)}
}

// Add inserts a tag.
func (t *TagSet) Add(tag string) {
	t.tags[tag] = true
}

// Has reports whether a tag is present.
func (t *TagSet) Has(tag string) bool {
	return t.tags[tag]
}

// List returns the tags.
func (t *TagSet) List() []string {
	out := make([]string, 0, len(t.tags))
	for tag := range t.tags {
		out = append(out, tag)
	}
	return out
}
