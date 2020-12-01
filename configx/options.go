package configx

import (
	"context"
	"io"
	"os"

	"github.com/knadh/koanf"

	"github.com/ory/x/watcherx"
)

type (
	OptionModifier func(p *Provider)
)

func WithContext(ctx context.Context) OptionModifier {
	return func(p *Provider) {
		p.ctx = ctx
	}
}

func WithImmutables(immutables []string) OptionModifier {
	return func(p *Provider) {
		p.immutables = immutables
	}
}

func WithWatcher(watcher func(event watcherx.Event, err error)) OptionModifier {
	return func(p *Provider) {
		p.onChanges = watcher
	}
}

func WithStderrValidationReporter() OptionModifier {
	return func(p *Provider) {
		p.onValidationError = func(k *koanf.Koanf, err error) {
			p.printHumanReadableValidationErrors(k, os.Stderr, err)
		}
	}
}

func WithStandardValidationReporter(w io.Writer) OptionModifier {
	return func(p *Provider) {
		p.onValidationError = func(k *koanf.Koanf, err error) {
			p.printHumanReadableValidationErrors(k, w, err)
		}
	}
}
