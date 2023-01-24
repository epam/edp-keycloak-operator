package adapter

import (
	"fmt"
)

type RealmEventConfig struct {
	AdminEventsDetailsEnabled bool     `json:"adminEventsDetailsEnabled"`
	AdminEventsEnabled        bool     `json:"adminEventsEnabled"`
	EnabledEventTypes         []string `json:"enabledEventTypes"`
	EventsEnabled             bool     `json:"eventsEnabled"`
	EventsExpiration          int      `json:"eventsExpiration"`
	EventsListeners           []string `json:"eventsListeners"`
}

func (a GoCloakAdapter) SetRealmEventConfig(realmName string, eventConfig *RealmEventConfig) error {
	rsp, err := a.startRestyRequest().
		SetBody(eventConfig).
		SetPathParams(map[string]string{keycloakApiParamRealm: realmName}).
		Put(a.buildPath(realmEventConfigPut))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("failed to set realm event config request: %w", err)
	}

	return nil
}
