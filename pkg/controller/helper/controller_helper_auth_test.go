package helper

import (
	"context"
	"testing"

	"github.com/go-resty/resty/v2"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/jarcoal/httpmock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateKeycloakClientFromLoginPassword(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(v1alpha1.AddToScheme(s))

	kc := v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			Secret: "test",
		},
	}
	lpSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: kc.Spec.Secret,
		},
		Data: map[string][]byte{
			"username": []byte("username"),
			"password": []byte("password"),
		},
	}
	httpmock.Activate()
	httpmock.RegisterResponder("POST", "/auth/realms/master/protocol/openid-connect/token",
		httpmock.NewStringResponder(200, `{}`))
	cl := fake.NewClientBuilder().WithRuntimeObjects(&kc, &lpSecret).Build()

	helper := MakeHelper(cl, s, nil)
	helper.restyClient = resty.New()
	httpmock.ActivateNonDefault(helper.restyClient.GetClient())

	_, err := helper.CreateKeycloakClientFromLoginPassword(context.Background(), &kc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}
