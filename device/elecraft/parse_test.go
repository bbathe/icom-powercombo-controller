package elecraft

import (
	"strings"
	"testing"
)

func TestParseResponses(t *testing.T) {
	t.Run("FLT", func(t *testing.T) {
		got, err := parseFLT("FLT0;")
		if err != nil || got != 0 {
			t.Fatalf("got %d err %v", got, err)
		}
		got, err = parseFLT("FLT3;")
		if err != nil || got != 3 {
			t.Fatalf("got %d err %v", got, err)
		}
		if _, err := parseFLT("FLT;"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("VSWR", func(t *testing.T) {
		got, err := parseVSWR("VSWR 1.50;")
		if err != nil || got != 1.5 {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("WS", func(t *testing.T) {
		got, err := parseWS("^WS010 000;")
		if err != nil || got != 10 {
			t.Fatalf("got %d err %v", got, err)
		}
	})

	t.Run("FL", func(t *testing.T) {
		got, err := parseFL("^FL00;")
		if err != nil || got != 0 {
			t.Fatalf("got %d err %v", got, err)
		}
	})

	t.Run("VI", func(t *testing.T) {
		v, a, err := parseVI("^VI505 123;")
		if err != nil {
			t.Fatal(err)
		}
		if v != 50.5 || a != 12.3 {
			t.Fatalf("got %v %v", v, a)
		}
		if _, _, err := parseVI("^VIbad;"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestReadMessageFromPort(t *testing.T) {
	got, err := readMessageFromPort(strings.NewReader("VSWR 1.50;noise"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "VSWR 1.50;" {
		t.Fatalf("got %q", got)
	}

	got, err = readMessageFromPort(strings.NewReader("^WS010 000;"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "^WS010 000;" {
		t.Fatalf("got %q", got)
	}
}
