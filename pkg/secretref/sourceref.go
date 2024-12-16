package secretref

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// GetValueFromSourceRef retries value from ConfigMap or Secret by SourceRef.
func GetValueFromSourceRef(ctx context.Context, sourceRef *common.SourceRef, namespace string, k8sClient client.Client) (string, error) {
	if sourceRef == nil {
		return "", nil
	}

	if sourceRef.ConfigMapKeyRef != nil {
		configMap := &corev1.ConfigMap{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      sourceRef.ConfigMapKeyRef.Name,
		}, configMap); err != nil {
			return "", fmt.Errorf("unable to get configmap: %w", err)
		}

		return configMap.Data[sourceRef.ConfigMapKeyRef.Key], nil
	}

	if sourceRef.SecretKeyRef != nil {
		secret := &corev1.Secret{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      sourceRef.SecretKeyRef.Name,
		}, secret); err != nil {
			return "", fmt.Errorf("unable to get secret: %w", err)
		}

		return string(secret.Data[sourceRef.SecretKeyRef.Key]), nil
	}

	return "", nil
}

// GetValueFromSourceRefOrVal retries value from ConfigMap or Secret or directly from value.
func GetValueFromSourceRefOrVal(ctx context.Context, sourceRef *common.SourceRefOrVal, namespace string, k8sClient client.Client) (string, error) {
	if sourceRef == nil {
		return "", nil
	}

	if sourceRef.SourceRef != nil {
		return GetValueFromSourceRef(ctx, sourceRef.SourceRef, namespace, k8sClient)
	}

	return sourceRef.Value, nil
}
