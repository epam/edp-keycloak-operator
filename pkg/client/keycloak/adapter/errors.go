package adapter

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v12"
)

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

func IsErrNotFound(err error) bool {
	errNotFound := NotFoundError("")

	if errors.As(err, &errNotFound) {
		return true
	}

	apiErr := gocloak.APIError{}
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusNotFound
	}

	apiErrp := &gocloak.APIError{}
	if errors.As(err, &apiErrp) {
		return apiErrp.Code == http.StatusNotFound
	}

	return false
}
