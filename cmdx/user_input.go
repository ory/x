package cmdx

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// asks for confirmation with the question string s and reads the answer
// pass nil to use os.Stdin and os.Stdout
func AskForConfirmation(s string, stdin io.Reader, stdout io.Writer) bool {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}

	reader := bufio.NewReader(stdin)

	for {
		_, err := fmt.Fprintf(stdout, "%s [y/n]: ", s)
		Must(err, "%s", err)

		response, err := reader.ReadString('\n')
		Must(err, "%s", err)

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
