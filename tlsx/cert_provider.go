package tlsx

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/watcherx"
	"github.com/pkg/errors"
)

type (
	Provider interface {
		SetCertificatesGenerator(CertificateGenerator) Provider
		GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
		LoadCertificates(certString, keyString string, certPath, keyPath string) error
	}

	provider struct {
		ctx    context.Context
		logger *logrusx.Logger

		certGen           CertificateGenerator
		certPath, keyPath string

		crts    []tls.Certificate
		crtsLck sync.RWMutex

		watchersCancel []func()
		fsEvent        watcherx.EventChannel
		watchersLck    sync.Mutex
	}

	CertificateGenerator func() []tls.Certificate
)

// NewProvider creates a tls.Certificate provider
func NewProvider(ctx context.Context, l *logrusx.Logger) Provider {
	p := &provider{
		ctx:    ctx,
		logger: l,

		fsEvent: make(watcherx.EventChannel, 1),
	}

	go p.watchCertificatesChanges()

	return p
}

func (p *provider) SetCertificatesGenerator(c CertificateGenerator) Provider {
	p.certGen = c
	return p
}

func (p *provider) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	p.crtsLck.RLock()
	defer p.crtsLck.RUnlock()

	if len(p.crts) == 0 {
		return nil, errors.New("No certificate loaded")
	}

	if hello != nil {
		for _, cert := range p.crts {
			if cert.Leaf != nil && cert.Leaf.VerifyHostname(hello.ServerName) == nil {
				return &cert, nil
			}
		}
	}
	return &p.crts[0], nil
}

func (p *provider) LoadCertificates(
	certString, keyString string,
	certPath, keyPath string,
) error {
	fromFiles := certPath != "" && keyPath != ""
	crts, err := Certificate(certString, keyString, certPath, keyPath)
	if err != nil && errors.Is(err, ErrNoCertificatesConfigured) && p.certGen != nil {
		crts = p.certGen()
		fromFiles = false
	} else if err != nil {
		return err
	}

	p.setCertificates(crts)

	if fromFiles {
		p.certPath = certPath
		p.keyPath = keyPath
		p.setWatcher(certPath, keyPath)
	} else {
		p.certPath = ""
		p.keyPath = ""
		p.deleteWatcher()
	}

	return nil
}

func (p *provider) setCertificates(crts []tls.Certificate) {
	p.crtsLck.Lock()
	p.crts = crts
	p.crtsLck.Unlock()
}

func (p *provider) setWatcher(certPath, keyPath string) {
	p.watchersLck.Lock()
	defer p.watchersLck.Unlock()

	p.deleteWatchersNoLock()

	if err := p.addWatcher(certPath); err != nil {
		p.logger.WithError(err).Fatalf("Could not create watcher with path: %s", certPath)
	}

	if err := p.addWatcher(keyPath); err != nil {
		p.logger.WithError(err).Fatalf("Could not create watcher with path: %s", keyPath)
	}

}

func (p *provider) addWatcher(fsPath string) error {
	ctx, cancel := context.WithCancel(p.ctx)
	_, err := watcherx.WatchFile(ctx, fsPath, p.fsEvent)
	if err != nil {
		cancel()
		return err
	}

	p.watchersCancel = append(p.watchersCancel, cancel)

	return nil
}

func (p *provider) deleteWatcher() {
	p.watchersLck.Lock()
	defer p.watchersLck.Unlock()

	p.deleteWatchersNoLock()
}

func (p *provider) deleteWatchersNoLock() {
	for _, cancel := range p.watchersCancel {
		cancel()
	}
	p.watchersCancel = nil
}

func (p *provider) watchCertificatesChanges() {
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case e, ok := <-p.fsEvent:
				if !ok {
					return
				}

				if err, isErr := e.(*watcherx.ErrorEvent); isErr {
					p.logger.WithError(err).Warningf("Error watching: %s", e.Source())
					continue
				}

				if _, isChange := e.(*watcherx.ChangeEvent); isChange {
					p.logger.Infof("TLS certificates changed, updating")
					if err := p.LoadCertificates("", "", p.certPath, p.keyPath); err != nil {
						p.logger.WithError(err).Errorf("Error in the new tls certificates")
						return
					}
				}
			}
		}
	}()
}
