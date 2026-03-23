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
func GetValueFromSourceRef(
	ctx context.Context,
	sourceRef *common.SourceRef,
	namespace string,
	k8sClient client.Client,
) (string, error) {
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

// GetValueFromSecretKeySelector retrieves a value from a Kubernetes Secret using a SecretKeySelector reference.
func GetValueFromSecretKeySelector(
	ctx context.Context,
	selector *common.SecretKeySelector,
	namespace string,
	k8sClient client.Client,
) (string, error) {
	if selector == nil {
		return "", nil
	}

	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      selector.Name,
	}, secret); err != nil {
		return "", fmt.Errorf("unable to get secret %s: %w", selector.Name, err)
	}

	val, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("secret %s does not contain key %s", selector.Name, selector.Key)
	}

	return string(val), nil
}

// GetValueFromSourceRefOrVal retries value from ConfigMap or Secret or directly from value.
func GetValueFromSourceRefOrVal(
	ctx context.Context,
	sourceRef *common.SourceRefOrVal,
	namespace string,
	k8sClient client.Client,
) (string, error) {
	if sourceRef == nil {
		return "", nil
	}

	if sourceRef.ConfigMapKeyRef != nil || sourceRef.SecretKeyRef != nil {
		return GetValueFromSourceRef(ctx, &sourceRef.SourceRef, namespace, k8sClient)
	}

	return sourceRef.Value, nil
}
