package viperx

import (
	"github.com/ory/x/stringslice"
	"github.com/ory/x/stringsx"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"time"
)

func d(l logrus.FieldLogger, new, old string) {
	l.Warnf("Configuration key %s is deprecated and will be removed in a future release. Use key %s instead!", new, old)
}

func GetFloat64(l logrus.FieldLogger, key string, fallback float64, deprecated ...string) float64 {
	v := viper.GetFloat64(key)
	for _, dk := range deprecated {
		if v != 0 {
			break
		}

		if vv := viper.GetFloat64(dk); vv != 0 {
			d(l, dk, key)
			v = vv
		}
	}

	if v == 0 {
		return fallback
	}

	return v
}

func GetInt(l logrus.FieldLogger, key string, fallback int, deprecated ...string) int {
	v := viper.GetInt(key)
	for _, dk := range deprecated {
		if v != 0 {
			break
		}

		if vv := viper.GetInt(dk); vv != 0 {
			d(l, dk, key)
			v = vv
		}
	}

	if v == 0 {
		return fallback
	}

	return v
}

func GetDuration(l logrus.FieldLogger, key string, fallback time.Duration, deprecated ...string) time.Duration {
	v := viper.GetDuration(key)
	for _, dk := range deprecated {
		if v != 0 {
			break
		}

		if vv := viper.GetDuration(dk); vv != 0 {
			d(l, dk, key)
			v = vv
		}
	}

	if v == 0 {
		return fallback
	}

	return v
}

func GetString(l logrus.FieldLogger, key string, fallback string, deprecated ...string) string {
	v := viper.GetString(key)
	for _, dk := range deprecated {
		if len(v) > 0 {
			break
		}

		if vv := viper.GetString(dk); len(vv) > 0 {
			d(l, dk, key)
			v = vv
		}
	}

	if len(v) == 0 {
		return fallback
	}

	return v
}

func GetBool(l logrus.FieldLogger, key string, deprecated ...string) bool {
	v := viper.GetBool(key)
	for _, dk := range deprecated {
		if v {
			break
		}

		if vv := viper.GetBool(dk); vv {
			d(l, dk, key)
			v = vv
		}
	}

	return v
}

func GetStringSlice(l logrus.FieldLogger, key string, fallback []string, deprecated ...string) []string {
	v := viper.GetStringSlice(key)
	for _, dk := range deprecated {
		if len(v) > 0 {
			break
		}

		if vv := viper.GetStringSlice(dk); len(vv) > 0 {
			d(l, dk, key)
			v = vv
		}
	}

	r := make([]string, 0, len(v))
	for _, s := range v {
		if len(s) == 0 {
			continue
		}

		if strings.Contains(s, ",") {
			r = append(r, stringslice.TrimSpaceEmptyFilter(stringsx.Splitx(s, ","))...)
		} else {
			r = append(r, s)
		}
	}

	if len(r) == 0 {
		return fallback
	}

	return r
}
