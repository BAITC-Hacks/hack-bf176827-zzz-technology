package assistant

import (
	"errors"

	"hackaton/pkg/llm"
)

var (
	ErrLLMDisabled     = llm.ErrDisabled
	ErrEmptyQuestion   = errors.New("empty question")
	ErrInvalidGID      = errors.New("invalid gid")
	ErrNodeNotFound    = errors.New("node not found")
	ErrClusterNotFound = errors.New("cluster not found")
	ErrUnknownTool     = errors.New("unknown tool")
)
