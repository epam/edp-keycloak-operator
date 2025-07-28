package helper

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/epam/edp-keycloak-operator/api/common"
)

type FailureCountable interface {
	GetFailureCount() int64
	SetFailureCount(count int64)
}

type StatusValue interface {
	GetStatus() string
	SetStatus(val string)
}

type StatusValueFailureCountable interface {
	FailureCountable
	StatusValue
}

func (h *Helper) SetFailureCount(fc FailureCountable) time.Duration {
	failures := fc.GetFailureCount()

	const timeoutSeconds = 10
	timeout := h.getTimeout(failures, timeoutSeconds*time.Second)
	failures += 1
	fc.SetFailureCount(failures)

	return timeout
}

func (h *Helper) getTimeout(factor int64, baseDuration time.Duration) time.Duration {
	return baseDuration * time.Duration(factor+1)
}

func IsFailuresUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(FailureCountable)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(FailureCountable)
	if !ok {
		return false
	}

	return oo.GetFailureCount() == no.GetFailureCount()
}

func SetSuccessStatus(el StatusValueFailureCountable) {
	el.SetStatus(common.StatusOK)
	el.SetFailureCount(0)
}
