package corsx

import (
	"net/http"
	"strconv"

	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ory/go-convenience/stringsx"
)

// ParseOptions parses CORS settings by using the `viper` framework. The following options are parsed:
//
//  - CORS_ALLOWED_CREDENTIALS
//  - CORS_DEBUG
//  - CORS_MAX_AGE
//  - CORS_ALLOWED_ORIGINS
//  - CORS_ALLOWED_METHODS
//  - CORS_ALLOWED_HEADERS
func ParseOptions() cors.Options {
	allowCredentials, err := strconv.ParseBool(viper.GetString("CORS_ALLOWED_CREDENTIALS"))
	if err != nil {
		allowCredentials = false
	}

	debug, err := strconv.ParseBool(viper.GetString("CORS_DEBUG"))
	if err != nil {
		debug = false
	}

	maxAge, err := strconv.Atoi(viper.GetString("CORS_MAX_AGE"))
	if err != nil {
		maxAge = 0
	}

	return cors.Options{
		AllowedOrigins:   stringsx.Splitx(viper.GetString("CORS_ALLOWED_ORIGINS"), ","),
		AllowedMethods:   stringsx.Splitx(viper.GetString("CORS_ALLOWED_METHODS"), ","),
		AllowedHeaders:   stringsx.Splitx(viper.GetString("CORS_ALLOWED_HEADERS"), ","),
		ExposedHeaders:   stringsx.Splitx(viper.GetString("CORS_EXPOSED_HEADERS"), ","),
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
		Debug:            debug,
	}
}

// Initialize starts the CORS middleware for a http.Handler if `viper.GetString("CORS_ENABLED") == "true"`.
func Initialize(h http.Handler, l logrus.FieldLogger) http.Handler {
	if viper.GetString("CORS_ENABLED") == "true" {
		l.Info("CORS is enabled")
		return cors.New(ParseOptions()).Handler(h)
	}

	return h
}
