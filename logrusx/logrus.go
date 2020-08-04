package logrusx

import (
	"github.com/sirupsen/logrus"

	"github.com/ory/viper"

	"github.com/ory/x/stringsx"
)

func newLogger(o *options) *logrus.Logger {
	l := o.l
	if l == nil {
		l = logrus.New()
	}

	if o.exitFunc != nil {
		l.ExitFunc = o.exitFunc
	}

	if o.level != nil {
		l.Level = *o.level
	} else {
		var err error
		l.Level, err = logrus.ParseLevel(stringsx.Coalesce(
			viper.GetString("log.level"),
			viper.GetString("LOG_LEVEL")))
		if err != nil {
			l.Level = logrus.InfoLevel
		}
	}

	if o.formatter != nil {
		l.Formatter = o.formatter
	} else {
		switch stringsx.Coalesce(o.format, viper.GetString("log.format"), viper.GetString("LOG_FORMAT")) {
		case "json":
			l.Formatter = &logrus.JSONFormatter{PrettyPrint: false}
		case "json_pretty":
			l.Formatter = &logrus.JSONFormatter{PrettyPrint: true}
		default:
			l.Formatter = &logrus.TextFormatter{
				DisableQuote:     true,
				DisableTimestamp: false,
				FullTimestamp:    true,
			}
		}
	}

	for _, hook := range o.hooks {
		l.AddHook(hook)
	}

	l.ReportCaller = o.reportCaller || l.IsLevelEnabled(logrus.TraceLevel)
	return l
}

type options struct {
	l             *logrus.Logger
	level         *logrus.Level
	formatter     logrus.Formatter
	format        string
	reportCaller  bool
	exitFunc      func(int)
	leakSensitive bool
	hooks         []logrus.Hook
}

type Option func(*options)

func ForceLevel(level logrus.Level) Option {
	return func(o *options) {
		o.level = &level
	}
}

func ForceFormatter(formatter logrus.Formatter) Option {
	return func(o *options) {
		o.formatter = formatter
	}
}

func ForceFormat(format string) Option {
	return func(o *options) {
		o.format = format
	}
}

func WithHook(hook logrus.Hook) Option {
	return func(o *options) {
		o.hooks = append(o.hooks, hook)
	}
}

func WithExitFunc(exitFunc func(int)) Option {
	return func(o *options) {
		o.exitFunc = exitFunc
	}
}

func ReportCaller(reportCaller bool) Option {
	return func(o *options) {
		o.reportCaller = reportCaller
	}
}

func UseLogger(l *logrus.Logger) Option {
	return func(o *options) {
		o.l = l
	}
}

func LeakSensitive() Option {
	return func(o *options) {
		o.leakSensitive = true
	}
}

func newOptions(opts []Option) *options {
	o := new(options)
	for _, f := range opts {
		f(o)
	}
	return o
}

// New creates a new logger with all the important fields set.
func New(name string, version string, opts ...Option) *Logger {
	o := newOptions(opts)
	return &Logger{
		leakSensitive: o.leakSensitive ||
			viper.GetBool("log.leak_sensitive_values") || viper.GetBool("LOG_LEAK_SENSITIVE_VALUES"),
		Entry: newLogger(o).WithFields(logrus.Fields{
			"audience": "application", "service_name": name, "service_version": version}),
	}
}

func NewAudit(name string, version string, opts ...Option) *Logger {
	return New(name, version, opts...).WithField("audience", "audit")
}
