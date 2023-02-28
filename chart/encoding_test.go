package chart

/*
func TestValidateChartEncoding(t *testing.T) {
	b64 := base64.RawURLEncoding.EncodeToString
	normalYearBytes := binary.BigEndian.AppendUint16(make([]byte, 0, 2), 2023)
	leapYearBytes := binary.BigEndian.AppendUint16(make([]byte, 0, 2), 2024)

	var (
		yearAndByte  = b64(append(normalYearBytes, 255))
		yearAndBytes = b64(append(normalYearBytes, bytes.Repeat([]byte{255, 127, 63, 11}, 8)...))

		yearAndBytesMax365Year = b64(
			append(normalYearBytes, append(bytes.Repeat([]byte{141}, 45), 0b11111000)...),
		)
		yearAndBytesMax366Year = b64(
			append(leapYearBytes, append(bytes.Repeat([]byte{141}, 45), 0b11111100)...),
		)

		leadingZeroByte = b64(append(normalYearBytes, 255, 128, 127, 11, 0))

		onesAfter365DaysNormalYear = b64(append(normalYearBytes,
			append(bytes.Repeat([]byte{141}, 45), 0b11111100)...,
		))

		onesAfter366DaysNormalYear = b64(append(leapYearBytes,
			append(bytes.Repeat([]byte{141}, 45), 0b11111110)...,
		))
	)

	var tests = []struct {
		encoded string
		valid   bool
	}{
		{encoded: "", valid: false},
		{encoded: "1" + yearAndByte, valid: false},
		{encoded: "1" + yearAndBytes, valid: false},

		{encoded: "0" + yearAndByte, valid: true},
		{encoded: "0" + yearAndBytes, valid: true},
		{encoded: "0" + yearAndBytesMax365Year, valid: true},
		{encoded: "0" + yearAndBytesMax366Year, valid: true},

		{encoded: "0" + leadingZeroByte, valid: false},
		{encoded: "0" + onesAfter365DaysNormalYear, valid: false},
		{encoded: "0" + onesAfter366DaysNormalYear, valid: false},

		{encoded: "0B-ff5m55EQgEAAAgQAAIEECBAgQECAgwEKAiQAgAACXMACAg0IgNAcBoR8CAAt-4", valid: true},
		{encoded: "0B-cAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAI", valid: true},
		{encoded: "0B-gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE", valid: true},
		{encoded: "0B-eA", valid: true},
		{encoded: "0B-fA", valid: true},
		{encoded: "0B-f_", valid: true},
		{encoded: "0B-f_gA", valid: true},
		{encoded: "0B-cAgA", valid: true},
		{encoded: "0B-cAkSAAgcgIADCgAoIGAKAQACJAgwpIEAECBAACEiiAEYAEgAAQIIEAgCDJ0AXA", valid: true},
	}

	for i, v := range tests {
		if ValidateChartEncoding([]byte(v.encoded)) != v.valid {
			t.Errorf("%v: '%v' unexpected: %v", i, v.encoded, !v.valid)
		}
	}
}
*/
