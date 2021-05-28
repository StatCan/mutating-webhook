## Mutating Webhook

This repository provides a simple library for easily deploying a [Kubernetes Admission Controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) that allows for easy mutation of objects.

## How To Use

### Implement `Mutator`

The `Mutator` interface is what needs to be implemented. It requires a single function: `Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error)`. In this function, implement the logic of your mutating webhook.

### Get A `MutatingWebhook` 

The `MutatingWebhook` interface returned by `NewMutatingWebhook(mutator Mutator, configs MutatingWebhookConfigs)` function is what is used to create the server. 

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

Once you instantiate the struct that implements the interface via the constructor, you can start the server!

#### ListenAndServe()

`ListenAndServe()` is how you'll start the server! It is a blocking function, so it's best to run it in a go routine.

#### Shutdown()
Once you're ready to stop the application, the `Shutdown()` function can be called.

### Endpoints

There are three endpoints that are available from the webserver:
- `/` - A welcome message is served at the root.
- `/mutate` - The `Mutate` function you implemented is served from this endpoint.
- `/_healthz` - A health endpoint for the Kubernetes Liveness Probe.
- `/_ready` - A readiness endpoint for the Kubernetes Readiness Probe.

## Example Code

```go
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	mutatingwebhook "github.com/statcan/mutating-webhook"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
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

	mutator := customMutator{
		// Your variables
	}

	mw, err := mutatingwebhook.NewMutatingWebhook(&mutator, mutatingwebhook.MutatingWebhookConfigs{
		// If you want to change defaults, update them here.
	})
	if err != nil {
		klog.Fatal(err)
	}
	defer klog.Info(mw.Shutdown(context.TODO()))

	// Make a channel for the error and launch the blocking function in its own thread
	errChan := make(chan error)
	go func() {
		errChan <- mw.ListenAndServe()
	}()

	// Create channel for interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt or for server error to exit
	select {
	case interrupt := <-c:
		klog.Info("received interrupt %d -- exiting", interrupt)
		return
	case err := <-errChan:
		klog.Error(err)
	}
}
```

## Dockerfile

A [Dockerfile](./Dockerfile) is supplied that can be used to build the Webhook quickly.

## Helm Chart

A Helm Chart is available in the [statcan/charts](https://github.com/statcan/charts/mutating-webhook). 
This is the preferred method of deployment which uses cert-manager's capabilities of generating certificates for TLS, a requirement for Admission Webhooks. 

## Testing

To test your webhook, you may use the tests available in `mutatingwebhook/mutatingwebhook_test.go` for inspiration in devising your own tests.
______________________

## Webhook Mutant

Ce répertoire fourni une base duquel un Mutating Webhook peut être facilement développer.
