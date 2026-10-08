package blendshape

import (
	"strconv"
	"strings"
)

// TargetError locates a failure on a specific blendshape target.
type TargetError struct {
	Target string
	Op     string
	Reason error
}

func (e *TargetError) Error() string {
	var b strings.Builder
	b.WriteString("blendshape target ")
	b.WriteString(strconv.Quote(e.Target))
	if e.Op != "" {
		b.WriteString(" on ")
		b.WriteString(e.Op)
	}
	b.WriteString(": ")
	if e.Reason != nil {
		b.WriteString(e.Reason.Error())
	}
	return b.String()
}

func (e *TargetError) Unwrap() error { return e.Reason }

// ErrorList aggregates per-target errors so one bad target never tears down
// the whole frame.
type ErrorList []error

func (el ErrorList) Error() string {
	if len(el) == 0 {
		return ""
	}
	var b strings.Builder
	for i, err := range el {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

// ErrOrNil returns nil when the list is empty.
func (el ErrorList) ErrOrNil() error {
	if len(el) == 0 {
		return nil
	}
	return el
}
