module github.com/statcan/mutating-webhook-base

go 1.15

require (
	// Add your needed dependencies
	istio.io/api v0.0.0-20210427161431-039c2e8d4bad
	
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v3 v3.0.0 // indirect
)
