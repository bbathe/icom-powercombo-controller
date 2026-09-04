package util

import (
	"testing"

	"github.com/bbathe/icom-powercombo-controller/config"
)

func TestBandFromFrequency(t *testing.T) {
	config.Bands = map[int]config.Band{
		20: {Low: 14000000, High: 14350000},
		40: {Low: 7000000, High: 7300000},
	}

	tests := []struct {
		name    string
		freq    int64
		want    int
		wantErr bool
	}{
		{name: "20m mid", freq: 14150000, want: 20},
		{name: "40m low edge", freq: 7000000, want: 40},
		{name: "40m high edge", freq: 7300000, want: 40},
		{name: "out of band", freq: 10000000, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BandFromFrequency(tt.freq)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got band %d, want %d", got, tt.want)
			}
		})
	}
}
