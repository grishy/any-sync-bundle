package lightfilenoderpc

func byteSlicesToStrings(byteSlices ...[]byte) []string {
	strs := make([]string, len(byteSlices))
	for i, b := range byteSlices {
		strs[i] = string(b)
	}
	return strs
}
