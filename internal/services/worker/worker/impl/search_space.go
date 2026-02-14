package impl

type SearchSpace struct {
	alphabet []byte
	base     uint64
	maxLen   int

	powers []uint64

	// prefix[l] = кол-во слов от 1 до l символов
	// prefix[0] = 0
	prefix []uint64
}

func NewSearchSpace(alphabet string, maxLen int) *SearchSpace {
	base := uint64(len(alphabet))

	powers := make([]uint64, maxLen+1)
	prefix := make([]uint64, maxLen+1)

	powers[0] = 1
	for i := 1; i <= maxLen; i++ {
		powers[i] = powers[i-1] * base
		prefix[i] = prefix[i-1] + powers[i]
	}

	return &SearchSpace{
		alphabet: []byte(alphabet),
		base:     base,
		maxLen:   maxLen,
		powers:   powers,
		prefix:   prefix,
	}
}

// TotalSize returns the total number of words in the search space
func (s *SearchSpace) TotalSize() uint64 {
	return s.prefix[s.maxLen]
}

// FillWord converts index to word and writes it to buf.
// Returns the length of the word, or 0 if index is out of range.
func (s *SearchSpace) FillWord(index uint64, buf []byte) int {
	length := 0
	for l := 1; l <= s.maxLen; l++ {
		if index < s.prefix[l] {
			length = l
			index -= s.prefix[l-1]
			break
		}
	}

	if length == 0 {
		return 0
	}

	for i := length - 1; i >= 0; i-- {
		buf[i] = s.alphabet[index%s.base]
		index /= s.base
	}

	return length
}
