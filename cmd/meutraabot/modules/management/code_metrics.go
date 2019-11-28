package management

import "fmt"

type CodeMetrics struct {
	Lines      int
	Words      int
	Characters int
}

func (m CodeMetrics) String() string {
	return fmt.Sprintf(
		"%v lines, %v words, and %v characters",
		m.Lines, m.Words, m.Characters,
	)
}
