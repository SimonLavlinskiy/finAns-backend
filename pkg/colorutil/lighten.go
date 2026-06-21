package colorutil

import (
	"fmt"
	"strconv"
	"strings"
)

// Lighten смешивает hex-цвет с белым (amount 0..1).
func Lighten(hex string, amount float64) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return hex
	}
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	nr := int(float64(r) + (255-float64(r))*amount + 0.5)
	ng := int(float64(g) + (255-float64(g))*amount + 0.5)
	nb := int(float64(b) + (255-float64(b))*amount + 0.5)
	return fmt.Sprintf("#%02X%02X%02X", nr, ng, nb)
}

func parseHex(hex string) (r, g, b int, ok bool) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 0, 0, 0, false
	}
	rv, err1 := strconv.ParseInt(h[0:2], 16, 0)
	gv, err2 := strconv.ParseInt(h[2:4], 16, 0)
	bv, err3 := strconv.ParseInt(h[4:6], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}
