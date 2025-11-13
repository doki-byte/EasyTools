package toml

import (
	"EasyTools/app/controller/connect/ssh/toml/internal/danger"
	"EasyTools/app/controller/connect/ssh/toml/internal/tracker"
	unstable2 "EasyTools/app/controller/connect/ssh/toml/unstable"
)

type strict struct {
	Enabled bool

	// Tracks the current key being processed.
	key tracker.KeyTracker

	missing []unstable2.ParserError
}

func (s *strict) EnterTable(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.key.UpdateTable(node)
}

func (s *strict) EnterArrayTable(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.key.UpdateArrayTable(node)
}

func (s *strict) EnterKeyValue(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.key.Push(node)
}

func (s *strict) ExitKeyValue(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.key.Pop(node)
}

func (s *strict) MissingTable(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.missing = append(s.missing, unstable2.ParserError{
		Highlight: keyLocation(node),
		Message:   "missing table",
		Key:       s.key.Key(),
	})
}

func (s *strict) MissingField(node *unstable2.Node) {
	if !s.Enabled {
		return
	}

	s.missing = append(s.missing, unstable2.ParserError{
		Highlight: keyLocation(node),
		Message:   "missing field",
		Key:       s.key.Key(),
	})
}

func (s *strict) Error(doc []byte) error {
	if !s.Enabled || len(s.missing) == 0 {
		return nil
	}

	err := &StrictMissingError{
		Errors: make([]DecodeError, 0, len(s.missing)),
	}

	for _, derr := range s.missing {
		derr := derr
		err.Errors = append(err.Errors, *wrapDecodeError(doc, &derr))
	}

	return err
}

func keyLocation(node *unstable2.Node) []byte {
	k := node.Key()

	hasOne := k.Next()
	if !hasOne {
		panic("should not be called with empty key")
	}

	start := k.Node().Data
	end := k.Node().Data

	for k.Next() {
		end = k.Node().Data
	}

	return danger.BytesRange(start, end)
}
