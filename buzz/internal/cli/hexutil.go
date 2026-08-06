package cli

// Small helpers shared by the channels/messages/agents command builders
// below. isHex64 is a thin boolean wrapper around validateHex64
// (reactions.go) for call sites that only need a yes/no check; arbitrary-
// length hex (commit SHAs, signatures) reuses isLowerHexOrUpper
// (git_common.go) rather than redefining the same scan.

func isHex64(s string) bool {
	_, err := validateHex64(s)
	return err == nil
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
