package pkg

import (
	"fmt"
)

// Range represents a range [Start, End)
type Range struct {
	Start uint64
	End   uint64
}

// SplitRange splits the total space [0, totalSize) into approximately equal ranges.
// Returns 'parts' ranges and an error if parts > totalSize.
func SplitRange(totalSize uint64, parts int) ([]Range, error) {
	if parts <= 0 {
		return []Range{}, nil
	}

	if totalSize == 0 {
		return nil, fmt.Errorf("split empty space")
	}

	if uint64(parts) > totalSize {
		return nil, fmt.Errorf("parts (%d) exceed total size (%d)", parts, totalSize)
	}

	ranges := make([]Range, parts)
	baseSize := totalSize / uint64(parts)
	remainder := totalSize % uint64(parts)

	var start uint64 = 0
	for i := 0; i < parts; i++ {
		size := baseSize
		if uint64(i) < remainder {
			size++ // distribute remainder among first workers
		}

		ranges[i] = Range{
			Start: start,
			End:   start + size,
		}
		start += size
	}

	return ranges, nil
}

func ToPtr[T any](v T) *T {
	return &v
}
