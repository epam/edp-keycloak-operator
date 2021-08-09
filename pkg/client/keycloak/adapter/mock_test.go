package adapter

import (
	"context"
	"testing"

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
