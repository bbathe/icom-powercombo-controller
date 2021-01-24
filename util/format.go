package util

import (
	"fmt"

	"github.com/bbathe/icom-powercombo-controller/config"
)

// BandFromFrequency returns the band corresponding to frequency
// band returned is the key to the config.Bands map
func BandFromFrequency(freq int64) (int, error) {
	for k, v := range config.Bands {
		if freq >= v.Low && freq <= v.High {
			return k, nil
		}
	}

	return 0, fmt.Errorf("out of band")
}
