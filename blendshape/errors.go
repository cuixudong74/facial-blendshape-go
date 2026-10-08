package blendshape

import "fmt"

// Issue 描述评估过程中被跳过的单个目标及原因，可定位到具体目标。
type Issue struct {
	Target string
	Reason string
}

func (i Issue) Error() string {
	return fmt.Sprintf("blendshape: target %q skipped: %s", i.Target, i.Reason)
}

// IssueList 聚合一帧内的所有问题，不中断整帧评估。
type IssueList []Issue

func (l IssueList) Error() string {
	if len(l) == 0 {
		return "blendshape: no issues"
	}
	s := fmt.Sprintf("blendshape: %d issue(s):", len(l))
	for _, i := range l {
		s += " [" + i.Error() + "]"
	}
	return s
}
