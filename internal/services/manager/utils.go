package manager

import "math"

// SplitAlphabet splits the alphabet into the specified number of parts.
// Uses ceiling division to determine part size.
// Duplicates are allowed when parts > alphabet length.
// Example: 39 chars / 10 parts = 4 chars per part (ceil(39/10) = 4)
// Example: 9 chars / 10 parts = some parts will be duplicates
func SplitAlphabet(alphabet string, parts int) []string {
	if parts <= 0 {
		return []string{}
	}

	length := len(alphabet)
	if length == 0 {
		return []string{}
	}

	// Calculate part size with ceiling division
	partSize := int(math.Ceil(float64(length) / float64(parts)))

	result := make([]string, parts)

	for i := 0; i < parts; i++ {
		start := i * partSize
		if start >= length {
			start = start % length
		}

		end := start + partSize
		if end > length {
			end = length
		}

		result[i] = alphabet[start:end]
	}

	return result
}
