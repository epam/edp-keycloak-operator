package adapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
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

	_, err := m.CreateClientScope(context.Background(), "foo", &cs)
	require.NoError(t, err)
}

func TestMock_OneLiners(t *testing.T) {
	m := Mock{}
	m.On("DeleteClientScope", "foo", "bar").Return(nil)
	err := m.DeleteClientScope(context.Background(), "foo", "bar")
	require.NoError(t, err)

	m.On("UpdateClientScope", "foo", "bar", &ClientScope{}).Return(nil)

	err = m.UpdateClientScope(context.Background(), "foo", "bar", &ClientScope{})
	require.NoError(t, err)

	m.On("UpdateClient", mock.Anything).Return(nil)
	err = m.UpdateClient(context.Background(), nil)
	require.NoError(t, err)
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
	err := m.SyncClientProtocolMapper(&dt, mappers, addOnly)
	require.NoError(t, err)
}

func TestMock_SyncServiceAccountRoles(t *testing.T) {
	m := Mock{}
	m.On("SyncServiceAccountRoles", "", "", []string{}, map[string][]string{}, false).Return(nil)
	err := m.SyncServiceAccountRoles("", "", []string{}, map[string][]string{}, false)
	require.NoError(t, err)
}

func TestMock_SetServiceAccountAttributes(t *testing.T) {
	m := Mock{}

	m.On("SetServiceAccountAttributes", "", "", map[string]string{}, false).Return(nil)
	err := m.SetServiceAccountAttributes("", "", map[string]string{}, false)
	require.NoError(t, err)
}

func TestMock_Component(t *testing.T) {
	var (
		m   Mock
		ctx context.Context
	)

	m.On("CreateComponent", "foo", testComponent()).Return(nil)
	err := m.CreateComponent(ctx, "foo", testComponent())
	require.NoError(t, err)

	m.On("UpdateComponent", "foo", testComponent()).Return(nil)
	err = m.UpdateComponent(ctx, "foo", testComponent())
	require.NoError(t, err)

	m.On("GetComponent", "foo", "bar").Return(testComponent(), nil).Once()
	_, err = m.GetComponent(ctx, "foo", "bar")
	require.NoError(t, err)

	m.On("GetComponent", "foo", "bar").Return(nil, errors.New("fatal"))
	if _, err := m.GetComponent(ctx, "foo", "bar"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("DeleteComponent", "foo", "bar").Return(nil)
	err = m.DeleteComponent(ctx, "foo", "bar")
	require.NoError(t, err)
}

func TestMock_GetClientScope(t *testing.T) {
	m := Mock{}
	m.On("GetClientScope", "scopeName", "realmName").Return(&ClientScope{}, nil).Once()
	_, err := m.GetClientScope("scopeName", "realmName")
	require.NoError(t, err)
	m.On("GetClientScope", "scopeName", "realmName").Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetClientScope("scopeName", "realmName"); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_PutClientScopeMapper(t *testing.T) {
	m := Mock{}
	m.On("PutClientScopeMapper", "realmName", "scopeID", &ProtocolMapper{}).Return(nil)
	err := m.PutClientScopeMapper("realmName", "scopeID", &ProtocolMapper{})
	require.NoError(t, err)
}

func TestMock_GetDefaultClientScopesForRealm(t *testing.T) {
	m := Mock{}
	m.On("GetDefaultClientScopesForRealm", "realm").Return([]ClientScope{}, nil).Once()
	_, err := m.GetDefaultClientScopesForRealm(context.Background(), "realm")
	require.NoError(t, err)

	m.On("GetDefaultClientScopesForRealm", "realm").Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetDefaultClientScopesForRealm(context.Background(), "realm"); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_GetClientScopeMappers(t *testing.T) {
	m := Mock{}
	m.On("GetClientScopeMappers", "realm", "scope").Return([]ProtocolMapper{}, nil).Once()
	_, err := m.GetClientScopeMappers(context.Background(), "realm", "scope")
	require.NoError(t, err)

	m.On("GetClientScopeMappers", "realm", "scope").
		Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetClientScopeMappers(context.Background(), "realm", "scope"); err == nil {
		t.Fatal("no error returned")
	}
}
