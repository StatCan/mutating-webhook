package mutatingwebhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	payload = corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "mutating-webhook-test",
					Image: "testimage",
				},
			},
		},
	}
)

type mute struct{}

func (m *mute) Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error) {

	admission := getAdmission()

	if request.UID != admission.Request.UID &&
		request.Kind != admission.Request.Kind {
		return v1.AdmissionResponse{}, fmt.Errorf("Pod object was not as expected!")
	}

	pod := corev1.Pod{}
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		return v1.AdmissionResponse{}, fmt.Errorf("Unable to unmarshal object")
	}

	if pod.Spec.Containers[0].Name != payload.Spec.Containers[0].Name &&
		pod.Spec.Containers[0].Image != payload.Spec.Containers[0].Image {
		return v1.AdmissionResponse{}, fmt.Errorf("Pod object was not as expected!")
	}

	return v1.AdmissionResponse{
		Allowed: true,
		Patch:   []byte("It has been mutated!"),
	}, nil
}

func TestIsCanServeAndShutdown(t *testing.T) {

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{})

	go mw.ListenAndMutate()
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	//Get Response with initial Certs
	resp, err := client.Get("https://localhost:8443/")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyString := string(bodyBytes)

	assert.Equal(t, "Hello from mutating-webhook! Mutation available on: /mutate", bodyString)

	mw.Shutdown(context.TODO())
}

func TestHealthEndpoint(t *testing.T) {

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{})

	go mw.ListenAndMutate()
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	//Get Response with initial Certs
	resp, err := client.Get("https://localhost:8443/_healthz")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyString := string(bodyBytes)

	assert.Equal(t, "ok", bodyString)

	mw.Shutdown(context.TODO())
}

func TestReadyEndpoint(t *testing.T) {

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{})

	go mw.ListenAndMutate()
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	//Get Response with initial Certs
	resp, err := client.Get("https://localhost:8443/_ready")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyString := string(bodyBytes)

	assert.Equal(t, "ok", bodyString)

	mw.Shutdown(context.TODO())
}

func TestCertReload(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), "mutatingwebhook_certreload_test")
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	// Cleanup
	defer os.RemoveAll(certDir)

	// Copy cert
	wd, err := os.Getwd()
	assert.NoError(t, err)

	err = copyFile(filepath.Join(wd, "certs", "tls.crt"), certFile)
	assert.NoError(t, err)

	err = copyFile(filepath.Join(wd, "certs", "tls.key"), keyFile)
	assert.NoError(t, err)

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})

	go mw.ListenAndMutate()
	// Wait for serving to start
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	//Get Response with initial Certs
	resp, err := client.Get("https://localhost:8443/")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Update certs
	err = copyFile(filepath.Join(wd, "certs", "tls2.crt"), certFile)
	assert.NoError(t, err)

	err = copyFile(filepath.Join(wd, "certs", "tls2.key"), keyFile)
	assert.NoError(t, err)

	//Wait for reload
	time.Sleep(100 * time.Millisecond)

	resp2, err := client.Get("https://localhost:8443/")
	assert.NoError(t, err)
	defer resp2.Body.Close()

	assert.NotEqualValues(t, resp.TLS.PeerCertificates, resp2.TLS.PeerCertificates)
	mw.Shutdown(context.TODO())
}

func TestCanMutate(t *testing.T) {

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{})

	go mw.ListenAndMutate()
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	var err error
	admission := getAdmission()
	admission.Request.Object.Object = &payload

	requestBody, err := json.Marshal(admission)
	assert.NoError(t, err)

	// Post an AdmissionReview to the mutate endpoint
	resp, err := client.Post("https://localhost:8443/mutate", "application/json", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	admisionReturned := v1.AdmissionReview{}
	err = json.Unmarshal(bodyBytes, &admisionReturned)
	assert.NoError(t, err)

	patchValue := string(admisionReturned.Response.Patch)
	assert.Equal(t, "It has been mutated!", patchValue)

	mw.Shutdown(context.TODO())
}

func TestRejectNonJSON(t *testing.T) {

	mw := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{})

	go mw.ListenAndMutate()
	defer mw.Shutdown(context.TODO())
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	var err error
	admission := getAdmission()
	admission.Request.Object.Object = &payload

	requestBody, err := json.Marshal(admission)
	assert.NoError(t, err)

	// Post an AdmissionReview to the mutate endpoint
	resp, err := client.Post("https://localhost:8443/mutate", "application/xml", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	body := string(bodyBytes)

	assert.Equal(t, "JSON is expected", body)
}

// Helper for getting a client that will accept self-signed certs.
func getClient() *http.Client {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Transport: transport}
}

// Helper function for filecopying
func copyFile(source, destination string) error {
	input, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(destination, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getAdmission() v1.AdmissionReview {
	admission := v1.AdmissionReview{
		Request: &v1.AdmissionRequest{
			UID: "This is unique!",
			Kind: metav1.GroupVersionKind{
				Group:   "mutating.statcan.gc.ca",
				Version: "UberAlpha",
				Kind:    "MutatorTest",
			},
		},
	}

	return admission
}
