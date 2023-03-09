package chart

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/bits"
	"strings"
	"time"
)

// 2B for year 46B for days of a year (366/8).
const maxLen = 2 + 46

var (
	errInvaldChartEndoding = errors.New("invalid chart encoding")
)

func Decode(enc string) ([]byte, error) {
	if len(enc) == 0 || enc[0] != '0' {
		return nil, errInvaldChartEndoding
	}

	enc = enc[1:]

	if base64.RawURLEncoding.DecodedLen(len(enc)) > maxLen {
		return nil, errInvaldChartEndoding
	}

	// Base64 decoder ignores \r and \n chars.
	if strings.ContainsAny(enc, "\r\n") {
		return nil, errInvaldChartEndoding
	}

	raw := make([]byte, maxLen)
	n, err := base64.RawURLEncoding.Decode(raw, []byte(enc))
	if err != nil || n < 3 {
		return nil, errInvaldChartEndoding
	}

	year := binary.BigEndian.Uint16(raw[:2])

	if raw := raw[:n]; len(raw) == maxLen {
		// Require that all trailing bits (above 365 or 366 days) are set to 0.
		daysInYear := time.Date(int(year)+1, time.January, 0, 0, 0, 0, 0, time.Local).YearDay()
		if bits.TrailingZeros8(raw[len(raw)-1]) < 8-daysInYear%8 {
			return nil, errInvaldChartEndoding
		}
	} else {
		// Require that the encodng does not have trailing zeros, so that
		// there is exacly one valid representation.
		if raw[len(raw)-1] == 0 {
			return nil, errInvaldChartEndoding
		}
	}

	nonZero := false
	for _, v := range raw[2:n] {
		if v != 0 {
			nonZero = true
			break
		}
	}

	// Require that at least one day bit is set to 1.
	if !nonZero {
		return nil, errInvaldChartEndoding
	}

	return raw, nil
}

func Encode(chart []byte) (string, error) {
	if len(chart) != maxLen {
		return "", errInvaldChartEndoding
	}
	n := len(chart) - 1
	for i := n; i >= 0; i-- {
		if chart[i] != 0 {
			n = i + 1
			break
		}
	}
	return "0" + base64.RawURLEncoding.EncodeToString(chart[:n]), nil
}
