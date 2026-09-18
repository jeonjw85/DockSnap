package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrInvalidTag = errors.New("invalid tag")
	tagRe         = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

type Tag struct {
	raw string
}

func NewTag(s string) (Tag, error) {
	if !tagRe.MatchString(s) {
		return Tag{}, fmt.Errorf("%q: %w", s, ErrInvalidTag)
	}
	return Tag{raw: s}, nil
}

func (t Tag) String() string {
	return t.raw
}

func (t Tag) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.raw)
}

func (t *Tag) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := NewTag(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
