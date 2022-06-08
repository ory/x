package tlsx

import (
	"context"
	"crypto/tls"
	"sync"

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
		ctx context.Context

		certGen           CertificateGenerator
		certPath, keyPath string

		crts    []tls.Certificate
		crtsLck sync.RWMutex

		watchersCancel []func()
		fsEvent        watcherx.EventChannel
		fsEvent2       watcherx.EventChannel
		watchersLck    sync.Mutex

		ev EventChannel
	}

	CertificateGenerator func() ([]tls.Certificate, error)

	EventChannel chan Event

	Event interface {
		String() string
	}

	ErrorEvent struct {
		error
	}

	ChangeEvent struct{}
)

func (e *ErrorEvent) String() string {
	return e.Error()
}

func (e *ChangeEvent) String() string {
	return "TLS Certificates changed, updating"
}

// NewProvider creates a tls.Certificate provider
func NewProvider(ctx context.Context, ev EventChannel) Provider {
	p := &provider{
		ctx: ctx,
		ev:  ev,
	}

	go p.watchCertificatesChanges()

	return p
}

func (p *provider) Event() EventChannel {
	return p.ev
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
	fromFiles, err := p.loadCertificates(certString, keyString, certPath, keyPath)
	if err != nil {
		return err
	}

	if fromFiles {
		p.certPath = certPath
		p.keyPath = keyPath
		return p.setWatcher(certPath, keyPath)
	} else {
		p.certPath = ""
		p.keyPath = ""
		p.deleteWatchers()
	}

	return nil
}

func (p *provider) loadCertificates(
	certString, keyString string,
	certPath, keyPath string,
) (fromFiles bool, err error) {
	fromFiles = certPath != "" && keyPath != ""
	crts, err := Certificate(certString, keyString, certPath, keyPath)
	if err != nil && errors.Is(err, ErrNoCertificatesConfigured) && p.certGen != nil {
		crts, err = p.certGen()
		if err != nil {
			return false, err
		}
		fromFiles = false
	} else if err != nil {
		return false, err
	}

	p.setCertificates(crts)

	return
}

func (p *provider) setCertificates(crts []tls.Certificate) {
	p.crtsLck.Lock()
	p.crts = crts
	p.crtsLck.Unlock()
}

func (p *provider) setWatcher(certPath, keyPath string) error {
	p.watchersLck.Lock()
	defer p.watchersLck.Unlock()

	p.deleteWatchersNoLock()

	if err := p.addFileWatcher(certPath, &p.fsEvent); err != nil && p.ev != nil {
		return err
	}

	if err := p.addFileWatcher(keyPath, &p.fsEvent2); err != nil && p.ev != nil {
		return err
	}

	return nil
}

func (p *provider) addFileWatcher(fsPath string, fsEvent *watcherx.EventChannel) error {
	ctx, cancel := context.WithCancel(p.ctx)

	*fsEvent = make(watcherx.EventChannel, 1)

	_, err := watcherx.WatchFile(ctx, fsPath, *fsEvent)
	if err != nil {
		cancel()
		return err
	}

	p.watchersCancel = append(p.watchersCancel, cancel)

	return nil
}

func (p *provider) deleteWatchers() {
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
				close(p.ev)
				return
			case e, ok := <-p.fsEvent:
				if !ok {
					return
				}

				p.handleEvent(e)
			}
		}
	}()
}

func (p *provider) handleEvent(e watcherx.Event) {
	switch ev := e.(type) {
	case *watcherx.ErrorEvent:
		if p.ev != nil {
			p.ev <- &ErrorEvent{error: ev}
		}

	case *watcherx.ChangeEvent:
		if p.ev != nil {
			p.ev <- &ChangeEvent{}
		}
		if _, err := p.loadCertificates("", "", p.certPath, p.keyPath); err != nil && p.ev != nil {
			p.ev <- &ErrorEvent{error: err}
		}
	}
}
