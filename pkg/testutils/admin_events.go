package testutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

const (
	// maxAdminEvents caps one event query. A steady-state window holds a handful of
	// events; a window at the cap means the query, not the controller, bounded the result.
	maxAdminEvents = 500
	// quietProbe is how long WritesDuring watches for an in-flight write before it opens
	// the window. Raise it if a "no writes" spec starts flaking.
	quietProbe = time.Second
)

// AdminEvents is one recorded window of Keycloak admin events. Gomega renders it as
// "OPERATION resourcePath" lines.
type AdminEvents []keycloakapi.AdminEventRepresentation

func (e AdminEvents) GomegaString() string {
	if len(e) == 0 {
		return "<none>"
	}

	lines := make([]string, 0, len(e))
	for _, ev := range e {
		lines = append(lines, fmt.Sprintf("%s %s",
			ptr.Deref(ev.OperationType, "?"), ptr.Deref(ev.ResourcePath, "?")))
	}

	return strings.Join(lines, "\n")
}

// AdminEventRecorder reports the Keycloak writes an action made. Keycloak logs one
// admin event per CREATE/UPDATE/DELETE, so an empty window means no write.
//
// Enable admin event storage before use, either by calling Enable or by setting
// spec.realmEventConfig on the realm CR under test.
//
// Scope the recorder to one resource when sibling specs share the realm.
type AdminEventRecorder struct {
	kc        *keycloakapi.KeycloakClient
	realm     string
	pathMatch string
}

func NewAdminEventRecorder(kc *keycloakapi.KeycloakClient, realm string) *AdminEventRecorder {
	return &AdminEventRecorder{kc: kc, realm: realm}
}

// Scoped returns a recorder that reports only events whose resourcePath contains
// pathSubstring. Pass the Keycloak-assigned resource ID: sibling specs share the realm.
func (r *AdminEventRecorder) Scoped(pathSubstring string) *AdminEventRecorder {
	scoped := *r
	scoped.pathMatch = pathSubstring

	return &scoped
}

// Enable turns on admin event storage for the realm, preserving the rest of the
// events config. Idempotent. The realm must already exist in Keycloak.
func (r *AdminEventRecorder) Enable(ctx context.Context) error {
	cfg, _, err := r.kc.Events.GetEventsConfig(ctx, r.realm)
	if err != nil {
		return fmt.Errorf("get events config: %w", err)
	}

	if cfg == nil {
		cfg = &keycloakapi.RealmEventsConfigRepresentation{}
	}

	cfg.AdminEventsEnabled = ptr.To(true)
	// Details store the full request body of every write. The recorder reads only
	// operationType and resourcePath, so details are load on a shared Keycloak.
	cfg.AdminEventsDetailsEnabled = ptr.To(false)

	if _, err = r.kc.Events.SetEventsConfig(ctx, r.realm, *cfg); err != nil {
		return fmt.Errorf("set events config: %w", err)
	}

	return nil
}

// WritesDuring waits for the realm to go quiet, opens an event window, runs action,
// waits settle, and returns the writes Keycloak recorded within this recorder's scope.
//
// The quiet wait keeps a reconcile still in flight from an earlier spec out of the
// window; without it that write is charged to action. Blocks up to LongWait waiting
// for quiet, and fails rather than open a window while writes are still landing.
//
// Pass a settle duration when action returns before the reconcile lands. Pass 0 when
// action itself blocks until Keycloak has converged.
//
// Assert at least once that a known write yields a non-empty result: an empty window
// is evidence only while the log is recording.
func (r *AdminEventRecorder) WritesDuring(
	ctx context.Context,
	settle time.Duration,
	action func(),
) (AdminEvents, error) {
	if err := r.quiesce(ctx); err != nil {
		return nil, err
	}

	action()

	if settle > 0 {
		time.Sleep(settle)
	}

	return r.recorded(ctx)
}

// quiesce clears the window and rechecks until one quietProbe passes with nothing
// recorded, leaving the window both empty and idle.
func (r *AdminEventRecorder) quiesce(ctx context.Context) error {
	deadline := time.Now().Add(LongWait)

	for {
		if err := r.clear(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for quiet realm %s: %w", r.realm, ctx.Err())
		case <-time.After(quietProbe):
		}

		events, err := r.recorded(ctx)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("realm %s still writing after %s:\n%s",
				r.realm, LongWait, events.GomegaString())
		}
	}
}

func (r *AdminEventRecorder) clear(ctx context.Context) error {
	if _, err := r.kc.Events.DeleteAdminEvents(ctx, r.realm); err != nil {
		return fmt.Errorf("clear admin events: %w", err)
	}

	return nil
}

func (r *AdminEventRecorder) recorded(ctx context.Context) (AdminEvents, error) {
	events, _, err := r.kc.Events.GetAdminEvents(ctx, r.realm, &keycloakapi.GetAdminEventsParams{
		Max: ptr.To(int32(maxAdminEvents)),
	})
	if err != nil {
		return nil, fmt.Errorf("get admin events: %w", err)
	}

	if r.pathMatch == "" {
		return events, nil
	}

	matched := make(AdminEvents, 0, len(events))

	for _, e := range events {
		// Realm-level writes carry no resourcePath and cannot be attributed. Report them
		// rather than drop them: an unattributable write inside the window is the write
		// amplification this recorder exists to catch.
		if e.ResourcePath == nil || strings.Contains(*e.ResourcePath, r.pathMatch) {
			matched = append(matched, e)
		}
	}

	return matched, nil
}
