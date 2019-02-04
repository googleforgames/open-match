package expbo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
)

// UnmarshalExponentialBackOff populates ExponentialBackOff structure parsing strings of format:
// "[InitInterval MaxInterval] *Multiplier ~RandomizationFactor <MaxElapsedTime"
//
// Example: "[0.250 30] *1.5 ~0.33 <7200"
func UnmarshalExponentialBackOff(s string, b *backoff.ExponentialBackOff) error {
	var (
		min, max, mult, rand, limit float64
		err                         error
	)

	for _, word := range strings.Split(strings.TrimSpace(s), " ") {
		switch {
		case word == "":
			continue
		case strings.HasPrefix(word, "["):
			min, err = strconv.ParseFloat(strings.TrimPrefix(word, "["), 64)
			if err != nil {
				return errors.New("cannot parse InitInterval value: " + err.Error())
			}
		case strings.HasSuffix(word, "]"):
			max, err = strconv.ParseFloat(strings.TrimSuffix(word, "]"), 64)
			if err != nil {
				return errors.New("cannot parse MaxInterval value: " + err.Error())
			}
		case strings.HasPrefix(word, "*"):
			mult, err = strconv.ParseFloat(strings.TrimPrefix(word, "*"), 64)
			if err != nil {
				return errors.New("cannot parse Multiplier value: " + err.Error())
			}
		case strings.HasPrefix(word, "~"):
			rand, err = strconv.ParseFloat(strings.TrimPrefix(word, "~"), 64)
			if err != nil {
				return errors.New("cannot parse RandomizationFactor value: " + err.Error())
			}
		case strings.HasPrefix(word, "<"):
			limit, err = strconv.ParseFloat(strings.TrimPrefix(word, "<"), 64)
			if err != nil {
				return errors.New("cannot parse MaxElapsedTime value: " + err.Error())
			}
		default:
			return fmt.Errorf(`unexpected word "%s"`, word)
		}
	}

	b.InitialInterval = time.Duration(min * float64(time.Second))
	b.MaxInterval = time.Duration(max * float64(time.Second))
	b.Multiplier = mult
	b.RandomizationFactor = rand
	b.MaxElapsedTime = time.Duration(limit * float64(time.Second))
	return nil
}
