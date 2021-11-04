package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

func TestMock_CreateClientScope(t *testing.T) {
	m := Mock{}
	cs := ClientScope{}
	m.On("CreateClientScope", "foo", &cs).Return("",
		errors.New("mock fatal")).Once()
	if _, err := m.CreateClientScope(context.Background(), "foo", &cs); err == nil {
		t.Fatal("no error returned")
	}

	m.On("CreateClientScope", "foo", &cs).Return("id1",
		nil)

	if _, err := m.CreateClientScope(context.Background(), "foo", &cs); err != nil {
		t.Fatal(err)
	}
}

func TestMock_OneLiners(t *testing.T) {
	m := Mock{}
	m.On("DeleteClientScope", "foo", "bar").Return(nil)
	if err := m.DeleteClientScope(context.Background(), "foo", "bar"); err != nil {
		t.Fatal(err)
	}

	m.On("UpdateClientScope", "foo", "bar", &ClientScope{}).Return(nil)
	if err := m.UpdateClientScope(context.Background(), "foo", "bar", &ClientScope{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_ExportToken(t *testing.T) {
	m := Mock{
		ExportTokenErr: errors.New("fatal"),
	}

	if _, err := m.ExportToken(); err == nil {
		t.Fatal("no error returned")
	}

}

func TestMock_SyncClientProtocolMapper(t *testing.T) {
	m := Mock{}
	dt := dto.Client{}
	mappers := []gocloak.ProtocolMapperRepresentation{{}}
	addOnly := false

	m.On("SyncClientProtocolMapper", &dt, mappers, addOnly).Return(nil)
	if err := m.SyncClientProtocolMapper(&dt, mappers, addOnly); err != nil {
		t.Fatal(err)
	}
}

func TestMock_SyncServiceAccountRoles(t *testing.T) {
	m := Mock{}
	m.On("SyncServiceAccountRoles", "", "", []string{}, map[string][]string{}, false).Return(nil)
	if err := m.SyncServiceAccountRoles("", "", []string{}, map[string][]string{}, false); err != nil {
		t.Fatal(err)
	}
}

func TestMock_SetServiceAccountAttributes(t *testing.T) {
	m := Mock{}

	m.On("SetServiceAccountAttributes", "", "", map[string]string{}, false).Return(nil)
	if err := m.SetServiceAccountAttributes("", "", map[string]string{}, false); err != nil {
		t.Fatal(err)
	}
}
