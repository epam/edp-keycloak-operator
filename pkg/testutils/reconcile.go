package testutils

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Settle bounds the wait for a forced reconcile to reach Keycloak. Raise it if a
	// "no writes" spec starts flaking.
	Settle = time.Second * 3
	// LongWait bounds an Eventually that spans a reconcile plus a Keycloak round-trip.
	LongWait = time.Second * 30
	// NudgeAnnotation carries the timestamp that forces a reconcile.
	NudgeAnnotation = "e2e.test/nudge"
	// pollInterval paces the status polling in ClearObservedGeneration.
	pollInterval = time.Millisecond * 250
)

// Nudge forces a reconcile of the object at key by stamping NudgeAnnotation.
// metadata.generation stays unchanged, so the controller takes its steady-state path.
// Pass an empty obj of the target type; Nudge fills it from the cluster.
func Nudge(ctx context.Context, c client.Client, key types.NamespacedName, obj client.Object) error {
	if err := c.Get(ctx, key, obj); err != nil {
		return fmt.Errorf("get %s: %w", key, err)
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[NudgeAnnotation] = time.Now().Format(time.RFC3339Nano)
	obj.SetAnnotations(annotations)

	if err := c.Update(ctx, obj); err != nil {
		return fmt.Errorf("update %s: %w", key, err)
	}

	return nil
}

// ClearObservedGeneration zeroes status.observedGeneration on the object at key, then
// waits until the controller restamps it to match metadata.generation.
//
// Mirrors a CR last reconciled by an operator build that predates observedGeneration
// tracking. The controller must write once even though Keycloak already matches spec:
// spec removals are invisible to a comparison that only checks spec-declared keys.
//
// Pass an empty obj of the target type; its kind comes from the client scheme.
func ClearObservedGeneration(
	ctx context.Context,
	c client.Client,
	key types.NamespacedName,
	obj client.Object,
) error {
	gvks, _, err := c.Scheme().ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("resolve kind for %T: %w", obj, err)
	}

	if len(gvks) == 0 {
		return fmt.Errorf("no kind registered for %T", obj)
	}

	gvk := gvks[0]

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, getErr := getUnstructured(ctx, c, key, gvk)
		if getErr != nil {
			return getErr
		}

		if setErr := unstructured.SetNestedField(current.Object, int64(0), "status", "observedGeneration"); setErr != nil {
			return fmt.Errorf("set observedGeneration on %s: %w", key, setErr)
		}

		return c.Status().Update(ctx, current)
	})
	if err != nil {
		return fmt.Errorf("clear observedGeneration on %s: %w", key, err)
	}

	return waitObservedGeneration(ctx, c, key, gvk)
}

func waitObservedGeneration(
	ctx context.Context,
	c client.Client,
	key types.NamespacedName,
	gvk schema.GroupVersionKind,
) error {
	err := wait.PollUntilContextTimeout(ctx, pollInterval, LongWait, false,
		func(ctx context.Context) (bool, error) {
			current, getErr := getUnstructured(ctx, c, key, gvk)
			if getErr != nil {
				return false, getErr
			}

			observed, found, nestedErr := unstructured.NestedInt64(current.Object, "status", "observedGeneration")
			if nestedErr != nil {
				return false, fmt.Errorf("read observedGeneration on %s: %w", key, nestedErr)
			}

			return found && observed == current.GetGeneration(), nil
		})
	if err != nil {
		return fmt.Errorf("wait for observedGeneration on %s: %w", key, err)
	}

	return nil
}

func getUnstructured(
	ctx context.Context,
	c client.Client,
	key types.NamespacedName,
	gvk schema.GroupVersionKind,
) (*unstructured.Unstructured, error) {
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(gvk)

	if err := c.Get(ctx, key, current); err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}

	return current, nil
}
