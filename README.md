## Mutating Webhook

This repository provides a simple library for easily deploying a [Kubernetes Admission Controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) that allows for easy mutation of objects.

## How To Use

### Implement `Mutator`

The `Mutator` interface is what needs to be implemented. It requires a single function: `Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error)`. In this function, implement the logic of your mutating webhook.

### `ListenAndMutate` Function

The `ListenAndMutate(mutator Mutator, configs MutatingWebhookConfigs)` function is what is used to start the webserver. 

#### Arguments

It requires two arguments:
- `mutator Mutator`: a reference to the `struct` that implements your `Mutate` function.
- `configs MutatingWebhookConfigs`: a reference to the configs you wish to pass to the webserver. Any `nil` values will use defaults.
  | Field          | Default           |
  | -------------- | ----------------- |
  | Addr           | ":8443"           |
  | ReadTimeout    | 10 * time.Second  |
  | WriteTimeout   | 10 * time.Second  |
  | MaxHeaderBytes | 0                 |
  | CertFilePath   | "./certs/tls.crt" |
  | KeyFilePath    | "./certs/tls.key" |

#### Endpoints

There are three endpoints that are available from the webserver:
- `/` - A welcome message is served at the root.
- `/mutate` - The `Mutate` function you implemented is served from this endpoint.
- `/_healthz` - A health endpoint for the Kubernetes Liveness and Readiness probes.

## Example Code

```go
package main

import (
	"encoding/json"

	"k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define the variables that you will need.
var (
// define the variables
)

// Initialize the variables.
func init() {
	// initialize the variables via arguments
	// flag.StringVar(&variable, "argument", "default-value", "Argument description.")
}

// Define the variables you need in the struct
type customMutator struct {
	// add variables you may need in your mutating code
}

// This is the function that will be called to mutate
func (cm *customMutator) Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error) {
	response := v1.AdmissionResponse{}

	// Default response
	response.Allowed = true
	response.UID = request.UID

	// Decode the object you are trying to mutate.
	// Here's an example with a Pod:

	// Decode the object
	// (an example for Pod)
	var err error
	//pod := v1.Pod{}
	// if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
	// 	return response, fmt.Errorf("unable to decode Pod %w", err)
	// }

	// Add the logic you wish to implement and create patches:
	patches := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/",
			"value": "",
		},
	}

	// If there are any patches, they will be appended to the
	if len(patches) > 0 {

		patchType := v1.PatchTypeJSONPatch
		response.PatchType = &patchType

		response.AuditAnnotations = map[string]string{
			// Add annotations to clearly denote that actions have
			// been performed on objects
		}

		response.Patch, err = json.Marshal(patches)
		if err != nil {
			return response, err
		}

		response.Result = &metav1.Status{
			Status: metav1.StatusSuccess,
		}
	}

	return response, nil
}

// Starts the webserver and serves the mutate function.
func main() {
	ListenAndMutate(
		&customMutator{},
		MutatingWebhookConfigs{},
	)
}
```

## Dockerfile

A [Dockerfile](./Dockerfile) is supplied that can be used to build the Webhook quickly.

## Helm Chart

A Helm Chart is available in the [statcan/charts](https://github.com/statcan/charts/mutating-webhook).

## Testing
______________________

## Webhook Mutant

Ce répertoire fourni une base duquel un Mutating Webhook peut être facilement développer.
