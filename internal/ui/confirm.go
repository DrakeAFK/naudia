package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ConfirmID(in io.Reader, out io.Writer, proposalID int64, message string) (bool, error) {
	fmt.Fprintln(out, message)
	fmt.Fprintf(out, "Type %d to confirm: ", proposalID)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return false, err
	}
	return strings.TrimSpace(line) == fmt.Sprintf("%d", proposalID), nil
}
