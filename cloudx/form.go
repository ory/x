package cloudx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	kratos "github.com/ory/kratos-client-go"
	"github.com/ory/x/cmdx"
)

func getLabel(attrs *kratos.UiNodeInputAttributes, node *kratos.UiNode) string {
	if attrs.Name == "password_identifier" {
		return fmt.Sprintf("%s: ", "Email")
	} else if node.Meta.Label != nil {
		return fmt.Sprintf("%s: ", node.Meta.Label.Text)
	} else if attrs.Label != nil {
		return fmt.Sprintf("%s: ", attrs.Label.Text)
	}
	return fmt.Sprintf("%s: ", attrs.Name)
}

type passwordReader func() ([]byte, error)

func renderForm(stdin *bufio.Reader, pwReader passwordReader, stdout io.Writer, ui kratos.UiContainer, method string, out interface{}) (err error) {
	for _, message := range ui.Messages {
		_, _ = fmt.Fprintf(stdout, "%s\n", message.Text)
	}

	for _, node := range ui.Nodes {
		for _, message := range node.Messages {
			_, _ = fmt.Fprintf(stdout, "%s\n", message.Text)
		}
	}

	values := json.RawMessage(`{}`)
	for _, node := range ui.Nodes {
		if node.Group != method {
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
					ok, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stdout)
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
			case "checkbox":
				result, err := cmdx.AskScannerForConfirmation(getLabel(attrs, &node), stdin, stdout)
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
					_, _ = fmt.Fprint(stdout, getLabel(attrs, &node))
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
					_, _ = fmt.Fprint(stdout, getLabel(attrs, &node))
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
