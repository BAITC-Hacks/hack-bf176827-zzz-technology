package graph

import "errors"

var (
	ErrNodeNotFound = errors.New("node not found")
	ErrInvalidDepth = errors.New("depth must be 1 or 2")
)
