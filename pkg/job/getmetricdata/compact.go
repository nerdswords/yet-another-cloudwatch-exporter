package getmetricdata

// compact iterates over a slice of pointers and deletes
// unwanted elements as per the keep function return value.
// The slice is modified in-place without copying elements.
func compact[T any](input []*T, keep func(el *T) bool) []*T {
	// move all elements that must be kept at the beginning
	i := 0
	for _, d := range input {
		if keep(d) {
			input[i] = d
			i++
		}
	}
	// nil out any left element
	for j := i; j < len(input); j++ {
		input[j] = nil
	}
	// set new slice length to allow released elements to be collected
	return input[:i]
}
