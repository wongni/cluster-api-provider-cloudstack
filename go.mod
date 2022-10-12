module sigs.k8s.io/cluster-api-provider-cloudstack

go 1.16

require (
	github.com/ReneKroon/ttlcache v1.7.0
	github.com/apache/cloudstack-go/v2 v2.13.0
	github.com/go-logr/logr v1.2.3
	github.com/golang/mock v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/smallfish/simpleyaml v0.1.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/text v0.3.7
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	k8s.io/klog/v2 v2.30.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.0
	sigs.k8s.io/controller-runtime v0.11.1
)

replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.0.0 // Indirect upgrade to address https://github.com/advisories/GHSA-w73w-5m7g-f7qc
