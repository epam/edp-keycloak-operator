package adapter

import "strings"

// SkipAlreadyExistsErr skips error if it is already exists error.
func SkipAlreadyExistsErr(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "409 Conflict") {
		return nil
	}

	return err
}
