package adapter

import (
	"github.com/pkg/errors"
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
		Put(a.basePath + realmEventConfigPut)

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "error during set realm event config request")
	}

	return nil
}
