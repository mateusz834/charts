package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math/bits"
	"time"
)

func ValidateChartEncoding(enc []byte) bool {
	if len(enc) == 0 || enc[0] != '0' {
		return false
	}

	enc = enc[1:]

	// 2B for year 46B for days of a year (366/8).
	const maxLen = 2 + 46
	if base64.RawURLEncoding.DecodedLen(len(enc)) > maxLen {
		return false
	}

	// Base64 decoder ignores \r and \n chars.
	if bytes.ContainsAny(enc, "\r\n") {
		return false
	}

	raw := make([]byte, maxLen)
	n, err := base64.RawURLEncoding.Decode(raw, enc)
	if err != nil || n < 3 {
		return false
	}

	raw = raw[:n]
	year := binary.BigEndian.Uint16(raw[:2])

	if len(raw) == maxLen {
		// Require that all trailing bits (above 365 or 366 days) are set to 0.
		daysInYear := time.Date(int(year)+1, time.January, 0, 0, 0, 0, 0, time.Local).YearDay()
		if bits.TrailingZeros8(raw[len(raw)-1]) < 8-daysInYear%8 {
			return false
		}
	} else {
		// Require that the encodng does not have trailing zeros, so that
		// there is exacly one valid representation.
		if raw[len(raw)-1] == 0 {
			return false
		}
	}

	return true
}
