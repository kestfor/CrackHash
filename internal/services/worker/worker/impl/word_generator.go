package impl

import (
	"iter"
)

type wordGenerator struct {
	maxLength int
	alphabet  []rune
}

func WordGenerator(maxLength int, alphabet string) *wordGenerator {
	//if maxLength <= 0 {
	//	return nil, fmt.Errorf("maxLength must be greater than 0")
	//}
	//
	//if len(alphabet) == 0 {
	//	return nil, fmt.Errorf("alphabet must not be empty")
	//}

	return &wordGenerator{
		maxLength: maxLength,
		alphabet:  []rune(alphabet),
	}
}

func (s *wordGenerator) Iterate() iter.Seq[string] {
	return func(yield func(string) bool) {
		if s == nil || s.maxLength == 0 || len(s.alphabet) == 0 {
			return
		}

		alphabetSize := len(s.alphabet)

		for wordLength := 1; wordLength <= s.maxLength; wordLength++ {
			letterIndexes := make([]int, wordLength)

			word := make([]rune, wordLength)

			for {
				for i, letterIndex := range letterIndexes {
					word[i] = s.alphabet[letterIndex]
				}

				if !yield(string(word)) {
					return
				}

				position := wordLength - 1

				for position >= 0 {
					letterIndexes[position]++

					if letterIndexes[position] < alphabetSize {
						break
					}

					letterIndexes[position] = 0
					position--
				}

				if position < 0 {
					break
				}
			}
		}
	}
}
