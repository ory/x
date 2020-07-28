package watcherx

import "io"

type (
	Event interface {
		Reader() io.Reader
		Source() string
	}
	source     string
	ErrorEvent struct {
		error
		source
	}
	ChangeEvent struct {
		data io.Reader
		source
	}
	RemoveEvent struct {
		source
	}
)

func (e *ErrorEvent) Reader() io.Reader {
	return nil
}

func (e source) Source() string {
	return string(e)
}

func (e *ChangeEvent) Reader() io.Reader {
	return e.data
}

func (e *RemoveEvent) Reader() io.Reader {
	return nil
}
