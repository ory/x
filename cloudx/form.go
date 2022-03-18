package cloudx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	cloud "github.com/ory/client-go"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	"github.com/ory/x/cmdx"
)

func getLabel(attrs *cloud.UiNodeInputAttributes, node *cloud.UiNode) string {
	if attrs.Name == "identifier" {
		return fmt.Sprintf("%s: ", "Email")
	} else if node.Meta.Label != nil {
		return fmt.Sprintf("%s: ", node.Meta.Label.Text)
	} else if attrs.Label != nil {
		return fmt.Sprintf("%s: ", attrs.Label.Text)
	}
	return fmt.Sprintf("%s: ", attrs.Name)
}

type passwordReader func() ([]byte, error)

func renderForm(stdin *bufio.Reader, pwReader passwordReader, stderr io.Writer, ui cloud.UiContainer, method string, out interface{}) (err error) {
	for _, message := range ui.Messages {
		_, _ = fmt.Fprintf(stderr, "%s\n", message.Text)
	}

	for _, node := range ui.Nodes {
		for _, message := range node.Messages {
			_, _ = fmt.Fprintf(stderr, "%s\n", message.Text)
		}
	}

	values := json.RawMessage(`{}`)
	for k := range ui.Nodes {
		node := ui.Nodes[k]
		if node.Group != method && node.Group != "default" {
			continue
		}

		switch node.Type {
		case "input":
			attrs := node.Attributes.UiNodeInputAttributes
			switch attrs.Type {
			case "button":
				continue
			case "submit":
				continue
			}

			if attrs.Name == "traits.consent.tos" {
				for {
					ok, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stderr)
					if err != nil {
						return err
					}
					if ok {
						break
					}
				}
				values, err = sjson.SetBytes(values, attrs.Name, time.Now().UTC().Format(time.RFC3339))
				if err != nil {
					return err
				}
				continue
			}

			switch attrs.Type {
			case "hidden":
				continue
			case "checkbox":
				result, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stderr)
				if err != nil {
					return err
				}

				values, err = sjson.SetBytes(values, attrs.Name, result)
				if err != nil {
					return err
				}
			case "password":
				var password string
				for password == "" {
					_, _ = fmt.Fprint(stderr, getLabel(attrs, &node))
					v, err := pwReader()
					if err != nil {
						return err
					}
					password = strings.ReplaceAll(string(v), "\n", "")
					fmt.Println("")
				}

				values, err = sjson.SetBytes(values, attrs.Name, password)
				if err != nil {
					return err
				}
			default:
				var value string
				for value == "" {
					_, _ = fmt.Fprint(stderr, getLabel(attrs, &node))
					v, err := stdin.ReadString('\n')
					if err != nil {
						return errors.Wrap(err, "failed to read from stdin")
					}
					value = strings.ReplaceAll(v, "\n", "")
				}

				values, err = sjson.SetBytes(values, attrs.Name, value)
				if err != nil {
					return err
				}
			}
		default:
			// Do nothing
		}
	}

	values, err = sjson.SetBytes(values, "method", method)
	if err != nil {
		return err
	}

	return errors.WithStack(json.NewDecoder(bytes.NewBuffer(values)).Decode(out))
}
