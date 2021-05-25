package chain

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Make(helper *helper.Helper, client client.Client, logger logr.Logger, factory keycloak.ClientFactory) Element {
	baseElement := BaseElement{
		State:  &State{},
		Helper: helper,
		Client: client,
		Logger: logger,
	}

	return &GetOrCreateRealmOwner{
		BaseElement: baseElement,
		next: &CreateAdapter{
			factory:     factory,
			BaseElement: baseElement,
			next: &PutClient{
				BaseElement: baseElement,
				next: &PutClientRole{
					BaseElement: baseElement,
					next: &PutRealmRole{
						BaseElement: baseElement,
						next: &PutClientScope{
							BaseElement: baseElement,
							next: &PutProtocolMappers{
								BaseElement: baseElement,
								next: &ServiceAccount{
									BaseElement: baseElement,
								},
							},
						},
					},
				},
			},
		},
	}
}
