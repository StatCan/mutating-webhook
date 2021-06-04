package mutatingwebhook

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
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

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
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
}

func TestHealthEndpoint(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
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
}

func TestReadyEndpoint(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
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
}

func writeCerts(certDir, name string) error {
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Statistics Canada"},
			Country:       []string{"CA"},
			Province:      []string{"ON"},
			Locality:      []string{"Ottawa"},
			CommonName:    name,
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},

		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  false,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	err = ioutil.WriteFile(certFile, certPEM.Bytes(), 0662)
	if err != nil {
		return err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	err = ioutil.WriteFile(keyFile, certPrivKeyPEM.Bytes(), 0662)
	if err != nil {
		return err
	}

	return nil
}

func TestCertReload(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	// Cleanup
	defer os.RemoveAll(certDir)

	err = writeCerts(certDir, "webhook1")
	assert.NoError(t, err)

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
	// Wait for serving to start
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	//Get Response with initial Certs
	resp, err := client.Get("https://localhost:8443/")
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Update certs
	err = writeCerts(certDir, "webhook2")
	assert.NoError(t, err)

	//Wait for reload
	time.Sleep(100 * time.Millisecond)

	resp2, err := client.Get("https://localhost:8443/")
	assert.NoError(t, err)
	defer resp2.Body.Close()

	assert.NotEqualValues(t, resp.TLS.PeerCertificates, resp2.TLS.PeerCertificates)
	assert.NotEqual(t, resp.TLS.PeerCertificates[0].Subject.CommonName, resp2.TLS.PeerCertificates[0].Subject.CommonName)
}

func TestCanMutate(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
	time.Sleep(100 * time.Millisecond)

	client := getClient()

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
}

func TestRejectNonJSON(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	admission := getAdmission()
	admission.Request.Object.Object = &payload

	requestBody, err := json.Marshal(admission)
	assert.NoError(t, err)

	// Post an AdmissionReview to the mutate endpoint
	resp, err := client.Post("https://localhost:8443/mutate", "content/xml", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	body := string(bodyBytes)

	assert.Equal(t, "JSON is expected", body)
}

func TestNoMediaType(t *testing.T) {

	// Setup cert location for testing
	certDir := filepath.Join(os.TempDir(), fmt.Sprintf("mutatingwebhook_certreload_test_%d", time.Now().Unix()))
	certFile := filepath.Join(certDir, "tls.cert")
	keyFile := filepath.Join(certDir, "tls.key")

	err := os.MkdirAll(certDir, 0770)
	assert.NoError(t, err)
	defer os.RemoveAll(certDir)

	writeCerts(certDir, "mutating-webhook")

	mw, err := NewMutatingWebhook(&mute{}, MutatingWebhookConfigs{
		CertFilePath: &certFile,
		KeyFilePath:  &keyFile,
	})
	assert.NoError(t, err)

	go mw.ListenAndServe()
	defer mw.Shutdown(context.TODO())
	time.Sleep(100 * time.Millisecond)

	client := getClient()

	admission := getAdmission()
	admission.Request.Object.Object = &payload

	requestBody, err := json.Marshal(admission)
	assert.NoError(t, err)

	// Post an AdmissionReview to the mutate endpoint
	resp, err := client.Post("https://localhost:8443/mutate", "", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	body := string(bodyBytes)

	assert.Equal(t, "mime: no media type", body)
}

// Helper for getting a client that will accept self-signed certs.
func getClient() *http.Client {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Transport: transport}
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
