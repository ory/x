package watcherx

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type (
	Event interface {
		Reader() io.Reader
		Source() string
		setSource(string)
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
	serialEventType string
	serialEvent     struct {
		Type   serialEventType `json:"type"`
		Data   []byte          `json:"data"`
		Source source          `json:"source"`
	}
)

const (
	serialTypeChange serialEventType = "change"
	serialTypeRemove serialEventType = "remove"
	serialTypeError  serialEventType = "error"
)

var unknownEventError = errors.New("unknown event type")

func (e *ErrorEvent) Reader() io.Reader {
	return nil
}

func (e *ErrorEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(serialEvent{
		Type:   serialTypeError,
		Data:   []byte(e.Error()),
		Source: e.source,
	})
}

func (e source) Source() string {
	return string(e)
}

func (e *source) setSource(nsrc string) {
	*e = source(nsrc)
}

func (e *ChangeEvent) Reader() io.Reader {
	return e.data
}

func (e *ChangeEvent) MarshalJSON() ([]byte, error) {
	var data []byte
	var err error
	if e.data != nil {
		data, err = ioutil.ReadAll(e.data)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return json.Marshal(serialEvent{
		Type:   serialTypeChange,
		Data:   data,
		Source: e.source,
	})
}

func (e *RemoveEvent) Reader() io.Reader {
	return nil
}

func (e *RemoveEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(serialEvent{
		Type:   serialTypeRemove,
		Source: e.source,
	})
}

func unmarshalEvent(data []byte) (Event, error) {
	var serialEvent serialEvent
	if err := json.Unmarshal(data, &serialEvent); err != nil {
		return nil, errors.WithStack(err)
	}
	switch serialEvent.Type {
	case serialTypeRemove:
		return &RemoveEvent{
			source: serialEvent.Source,
		}, nil
	case serialTypeChange:
		return &ChangeEvent{
			data:   bytes.NewBuffer(serialEvent.Data),
			source: serialEvent.Source,
		}, nil
	case serialTypeError:
		return &ErrorEvent{
			error:  errors.New(string(serialEvent.Data)),
			source: serialEvent.Source,
		}, nil
	}
	return nil, unknownEventError
}
