package helper

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// UpdateConnectionStatus maps connectErr onto the connected flag and status value, then writes
// obj's status. An unchanged status is not written: no resourceVersion bump, no watch event.
func UpdateConnectionStatus(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	connected *bool,
	value *string,
	connectErr error,
) error {
	log := ctrl.LoggerFrom(ctx)

	if connectErr != nil {
		log.Error(connectErr, "Unable to connect to Keycloak")
	}

	newConnected := connectErr == nil

	newValue := common.StatusOK
	if connectErr != nil {
		newValue = connectErr.Error()
	}

	if *connected == newConnected && *value == newValue {
		log.Info("Connection status hasn't been changed", "status", *connected)

		return nil
	}

	log.Info("Connection status has been changed", "from", *connected, "to", newConnected)

	*connected = newConnected
	*value = newValue

	if err := k8sClient.Status().Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Info("Status has been updated", "connected", *connected, "value", *value)

	return nil
}
