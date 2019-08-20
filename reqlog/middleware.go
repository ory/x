package negronilogrus

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Middleware is a middleware handler that logs the request as it goes in and the response as it goes out.
type Middleware struct {
	// Logger is the log.Logger instance used to log messages with the Logger middleware
	Logger *logrus.Logger
	// Name is the name of the application as recorded in latency metrics
	Name   string
	Before func(*logrus.Entry, *http.Request, string) *logrus.Entry
	After  func(*logrus.Entry, negroni.ResponseWriter, time.Duration, string) *logrus.Entry

	logStarting bool

	clock timer

	logLevel logrus.Level

	// Silence log for specific URL paths
	silencePaths map[string]bool

	sync.RWMutex
}

// NewMiddleware returns a new *Middleware, yay!
func NewMiddleware() *Middleware {
	return NewCustomMiddleware(logrus.InfoLevel, &logrus.TextFormatter{}, "web")
}

// NewCustomMiddleware builds a *Middleware with the given level and formatter
func NewCustomMiddleware(level logrus.Level, formatter logrus.Formatter, name string) *Middleware {
	log := logrus.New()
	log.Level = level
	log.Formatter = formatter

	return &Middleware{
		Logger: log,
		Name:   name,
		Before: DefaultBefore,
		After:  DefaultAfter,

		logLevel:    logrus.InfoLevel,
		logStarting: true,
		clock:       &realClock{},
		silencePaths: map[string]bool{},
	}
}

// NewMiddlewareFromLogger returns a new *Middleware which writes to a given logrus logger.
func NewMiddlewareFromLogger(logger *logrus.Logger, name string) *Middleware {
	return &Middleware{
		Logger: logger,
		Name:   name,
		Before: DefaultBefore,
		After:  DefaultAfter,

		logLevel:    logrus.InfoLevel,
		logStarting: true,
		clock:       &realClock{},
		silencePaths: map[string]bool{},
	}
}

// SetLogStarting accepts a bool to control the logging of "started handling
// request" prior to passing to the next middleware
func (m *Middleware) SetLogStarting(v bool) {
	m.logStarting = v
}

// ExcludePaths adds new URL paths to be ignored during logging. The URL u is parsed, hence the returned error
func (m *Middleware) ExcludePaths(paths ...string) *Middleware {
	for _, path := range paths {
		m.Lock()
		m.silencePaths[path] = true
		m.Unlock()
	}
	return nil
}

func (m *Middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if m.Before == nil {
		m.Before = DefaultBefore
	}

	if m.After == nil {
		m.After = DefaultAfter
	}

	logLevel := m.logLevel
	m.RLock()
	if _, ok := m.silencePaths[r.URL.Path]; ok {
		logLevel = logrus.DebugLevel
	}
	m.RUnlock()

	start := m.clock.Now()

	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	entry := logrus.NewEntry(m.Logger)

	if reqID := r.Header.Get("X-Request-Id"); reqID != "" {
		entry = entry.WithField("request_id", reqID)
	}

	entry = m.Before(entry, r, remoteAddr)

	if m.logStarting {
		entry.Log(logLevel, "started handling request")
	}

	next(rw, r)

	latency := m.clock.Since(start)
	res := rw.(negroni.ResponseWriter)

	m.After(entry, res, latency, m.Name).Log(logLevel, "completed handling request")
}

// BeforeFunc is the func type used to modify or replace the *logrus.Entry prior
// to calling the next func in the middleware chain
type BeforeFunc func(*logrus.Entry, *http.Request, string) *logrus.Entry

// AfterFunc is the func type used to modify or replace the *logrus.Entry after
// calling the next func in the middleware chain
type AfterFunc func(*logrus.Entry, negroni.ResponseWriter, time.Duration, string) *logrus.Entry

// DefaultBefore is the default func assigned to *Middleware.Before
func DefaultBefore(entry *logrus.Entry, req *http.Request, remoteAddr string) *logrus.Entry {
	return entry.WithFields(logrus.Fields{
		"request": req.RequestURI,
		"method":  req.Method,
		"remote":  remoteAddr,
	})
}

// DefaultAfter is the default func assigned to *Middleware.After
func DefaultAfter(entry *logrus.Entry, res negroni.ResponseWriter, latency time.Duration, name string) *logrus.Entry {
	return entry.WithFields(logrus.Fields{
		"status":                                res.Status(),
		"text_status":                           http.StatusText(res.Status()),
		"took":                                  latency,
		fmt.Sprintf("measure#%s.latency", name): latency.Nanoseconds(),
	})
}
