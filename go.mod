module github.com/epam/keycloak-operator

go 1.14

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37
)

require (
	github.com/Nerzal/gocloak/v8 v8.1.1
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/epam/edp-component-operator v0.1.1-0.20210413101042-1d8f823f27cc
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.19.3
	github.com/go-resty/resty/v2 v2.3.0
	github.com/google/uuid v1.1.2
	github.com/jarcoal/httpmock v1.0.8
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.1.2
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	sigs.k8s.io/controller-runtime v0.8.3
)