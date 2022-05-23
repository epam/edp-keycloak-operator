package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v10"
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

func TestMock_Component(t *testing.T) {
	var (
		m   Mock
		ctx context.Context
	)

	m.On("CreateComponent", "foo", testComponent()).Return(nil)
	if err := m.CreateComponent(ctx, "foo", testComponent()); err != nil {
		t.Fatal(err)
	}

	m.On("UpdateComponent", "foo", testComponent()).Return(nil)
	if err := m.UpdateComponent(ctx, "foo", testComponent()); err != nil {
		t.Fatal(err)
	}

	m.On("GetComponent", "foo", "bar").Return(testComponent(), nil).Once()
	if _, err := m.GetComponent(ctx, "foo", "bar"); err != nil {
		t.Fatal(err)
	}

	m.On("GetComponent", "foo", "bar").Return(nil, errors.New("fatal"))
	if _, err := m.GetComponent(ctx, "foo", "bar"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("DeleteComponent", "foo", "bar").Return(nil)
	if err := m.DeleteComponent(ctx, "foo", "bar"); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetClientScope(t *testing.T) {
	m := Mock{}
	m.On("GetClientScope", "scopeName", "realmName").Return(&ClientScope{}, nil).Once()
	if _, err := m.GetClientScope("scopeName", "realmName"); err != nil {
		t.Fatal(err)
	}
	m.On("GetClientScope", "scopeName", "realmName").Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetClientScope("scopeName", "realmName"); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_PutClientScopeMapper(t *testing.T) {
	m := Mock{}
	m.On("PutClientScopeMapper", "realmName", "scopeID", &ProtocolMapper{}).Return(nil)
	if err := m.PutClientScopeMapper("realmName", "scopeID", &ProtocolMapper{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetDefaultClientScopesForRealm(t *testing.T) {
	m := Mock{}
	m.On("GetDefaultClientScopesForRealm", "realm").Return([]ClientScope{}, nil).Once()
	if _, err := m.GetDefaultClientScopesForRealm(context.Background(), "realm"); err != nil {
		t.Fatal(err)
	}

	m.On("GetDefaultClientScopesForRealm", "realm").Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetDefaultClientScopesForRealm(context.Background(), "realm"); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_GetClientScopeMappers(t *testing.T) {
	m := Mock{}
	m.On("GetClientScopeMappers", "realm", "scope").Return([]ProtocolMapper{}, nil).Once()
	if _, err := m.GetClientScopeMappers(context.Background(), "realm", "scope"); err != nil {
		t.Fatal(err)
	}

	m.On("GetClientScopeMappers", "realm", "scope").
		Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetClientScopeMappers(context.Background(), "realm", "scope"); err == nil {
		t.Fatal("no error returned")
	}
}
