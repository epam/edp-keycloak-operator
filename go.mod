module github.com/epam/edp-keycloak-operator

go 1.14

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37
	k8s.io/api => k8s.io/api v0.20.7-rc.0
)

require (
	github.com/Nerzal/gocloak/v8 v8.1.1
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/epam/edp-common v0.0.0-20211124100535-e54dcdf42879
	github.com/epam/edp-component-operator v0.1.1-0.20210712140516-09b8bb3a4cff
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.19.3
	github.com/go-resty/resty/v2 v2.3.0
	github.com/google/uuid v1.1.2
	github.com/jarcoal/httpmock v1.0.8
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.1.2
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.21.0-rc.0
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/client-go v0.20.2
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/controller-runtime v0.8.3
)
