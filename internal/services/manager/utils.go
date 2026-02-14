package manager

func SplitAlphabet(alphabet string, parts int) []string {
	length := len(alphabet)
	partSize := length / parts
	remainder := length % parts

	result := make([]string, parts)
	start := 0

	for i := 0; i < parts; i++ {
		end := start + partSize
		if i < remainder {
			end++
		}
		result[i] = alphabet[start:end]
		start = end
	}

	return result
}
