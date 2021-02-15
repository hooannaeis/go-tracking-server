package processfactory

import (
	"fmt"
	"strings"
)

// DoProcess will return the processed value
// you can directly assign to the "where" of the event
func DoProcess(what string, how string) string {
	fmt.Printf("processing %s with %q\n", what, how)
	processedValue := "no such function: " + how
	switch how {
	case "getAnonymizedIP":
		processedValue = getAnonymizedIP(what)
	case "addToEvent":
		processedValue = what
		//@todo: add more process-options
	}
	return processedValue
}

func getAnonymizedIP(inputIP string) string {
	ipV4Identifier := "."
	ipV6Identifier := ":"
	anonymizedIP := ""
	isIPV4 := strings.Contains(inputIP, ipV4Identifier)
	if isIPV4 {
		parts := strings.Split(inputIP, ipV4Identifier)
		anonymizedIP = strings.Join(parts[0:len(parts)-1], ipV4Identifier) + ipV4Identifier + "0"
		return anonymizedIP
	}
	parts := strings.Split(inputIP, ipV6Identifier)
	numberOfBlocks := len(parts)
	anonymizedIP = strings.Join(parts[0:numberOfBlocks-5], ipV6Identifier) + ipV6Identifier + ipV6Identifier
	return anonymizedIP
}
