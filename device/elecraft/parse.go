package elecraft

import (
	"fmt"
	"strconv"
	"strings"
)

func parseFLT(msg string) (int, error) {
	s := strings.TrimPrefix(msg, "FLT")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}
	fault, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return fault, nil
}

func parseVSWR(msg string) (float64, error) {
	s := strings.TrimPrefix(msg, "VSWR ")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}
	return strconv.ParseFloat(s, 64)
}

func parseWS(msg string) (int, error) {
	s := strings.TrimPrefix(msg, "^WS")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}
	ss := strings.Split(s, " ")
	return strconv.Atoi(ss[0])
}

func parseFL(msg string) (int, error) {
	s := strings.TrimPrefix(msg, "^FL")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, fmt.Errorf("no serial response")
	}
	return strconv.Atoi(s)
}

func parseVI(msg string) (float64, float64, error) {
	s := strings.TrimPrefix(msg, "^VI")
	s = strings.TrimSuffix(s, ";")
	if len(s) == 0 {
		return 0, 0, fmt.Errorf("no serial response")
	}

	ss := strings.Split(s, " ")
	if len(ss) < 2 || len(ss[0]) < 3 || len(ss[1]) < 3 {
		return 0, 0, fmt.Errorf("invalid VI response %q", msg)
	}
	v := ss[0][:2] + "." + ss[0][2:]
	a := ss[1][:2] + "." + ss[1][2:]

	volts, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, 0, err
	}
	amps, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return 0, 0, err
	}
	return volts, amps, nil
}
