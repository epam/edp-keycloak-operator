package chain

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Make(scheme *runtime.Scheme, client client.Client, logger logr.Logger) Element {
	baseElement := BaseElement{
		scheme: scheme,
		Client: client,
		Logger: logger,
	}

	return &PutClient{
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
	}
}
