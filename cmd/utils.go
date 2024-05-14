package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

func printlnStdErr(a ...any) {
	fmt.Fprintln(os.Stderr, a...)
}

func PrintfStdErr(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func exitWithErrorMsg(error string) {
	printCheckRecords()
	printlnStdErr(error)
	os.Exit(1)
}

func exitWithErrorMsgf(format string, a ...any) {
	printCheckRecords()
	PrintfStdErr(format, a...)
	os.Exit(1)
}

func printCheckRecords() {
	if len(checkRecords) == 0 {
		return
	}

	printlnStdErr("\nReports:")

	sort.Slice(checkRecords, func(i, j int) bool {
		left := checkRecords[i]
		right := checkRecords[j]
		if left.fatal && !right.fatal {
			return true
		}
		if !left.fatal && right.fatal {
			return false
		}
		return left.addedNo < right.addedNo
	})

	for idx, record := range checkRecords {
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%2d. ", idx+1))
		if record.fatal {
			sb.WriteString("FATAL: ")
		}
		sb.WriteString(record.message)
		if record.suggest != "" {
			sb.WriteString(fmt.Sprintf("\n > %s", record.suggest))
		}
		printlnStdErr(sb.String())
	}
}

var regexPeerPlus = regexp.MustCompile(`^[a-f\d]{40}@(([^:]+)|(\[[a-f\d]*(:+[a-f\d]+)+])):\d{1,5}(,[a-f\d]{40}@(([^:]+)|(\[[a-f\d]*(:+[a-f\d]+)+])):\d{1,5})+$`)

func isValidPeer(peer string) bool {
	return regexPeerPlus.MatchString(peer)
}
