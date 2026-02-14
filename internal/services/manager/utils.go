package manager

import "fmt"

// SearchSpaceSize calculates the total number of words in the search space.
// For alphabet of size n and maxLength l, total = n + n^2 + ... + n^l = n*(n^l - 1)/(n-1)
func SearchSpaceSize(alphabetSize int, maxLength int) uint64 {
	if alphabetSize <= 0 || maxLength <= 0 {
		return 0
	}

	base := uint64(alphabetSize)
	var total uint64 = 0
	var power uint64 = 1

	for i := 1; i <= maxLength; i++ {
		power *= base
		total += power
	}

	return total
}

// Range represents a range [Start, End) for a worker to process
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
		return nil, fmt.Errorf("cannot split empty space")
	}

	if uint64(parts) > totalSize {
		return nil, fmt.Errorf("parts (%d) cannot exceed total size (%d)", parts, totalSize)
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
