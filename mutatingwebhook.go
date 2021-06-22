package mutatingwebhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/net/http2"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
)

const internalServerError = "an internal server error has occurred"

// Based on:
// https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec
// https://github.com/alex-leonhardt/k8s-mutate-webhook

// The Mutator interface is what is implemented to
// pass the mutation logic to the webserver.
type Mutator interface {
	Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error)
}

// The basic functions that are needed from a Server.
// Basic but opionated.
type MutatingWebhook interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// A function meant to handle the root of the server.
// For simpler debugging.
func (mw *mutatingWebhook) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from mutating-webhook! Mutation available on: /mutate")
}

// A Health endpoint to simplify use within an orchestrated environment.
func (mw *mutatingWebhook) handleHealthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

// handleMutate is what wraps the Mutator and serves the logic. of the Mutator.
func (mw *mutatingWebhook) handleMutate(w http.ResponseWriter, r *http.Request) {

	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	// Make sure content type is correct
	if contentType != "application/json" {
		klog.Warningf("contentType was not application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		fmt.Fprintf(w, "JSON is expected")
		return
	}

	// Decode the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", internalServerError)
		return
	}
	defer r.Body.Close()

	klog.V(5).Infof("request body:\n%s", body)

	// Attempt to get the AdmissionReview the request
	admissionReview := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", internalServerError)
		return
	}

	// Evaluate/Mutate the AdmissionRequest.
	response, err := mw.mutator.Mutate(*admissionReview.Request)
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", internalServerError)
		return
	}

	reviewResponse := v1.AdmissionReview{
		Response: &response,
	}

	if body, err = json.Marshal(reviewResponse); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", internalServerError)
		return
	}

	klog.V(5).Infof("response body:\n%s", string(body))

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

type mutatingWebhook struct {
	mutator     Mutator
	configs     MutatingWebhookConfigs
	server      *http.Server
	fileWatcher *fsnotify.Watcher
}

// Creates a MutatingWebhook server.
func NewMutatingWebhook(
	mutator Mutator,
	configs MutatingWebhookConfigs,
) (MutatingWebhook, error) {

	configs = setDefaults(configs)
	mux := http.NewServeMux()
	server := http.Server{
		Addr:           *configs.Addr,
		Handler:        mux,
		ReadTimeout:    *configs.ReadTimeout,
		WriteTimeout:   *configs.WriteTimeout,
		MaxHeaderBytes: *configs.MaxHeaderBytes,
	}

	mw := &mutatingWebhook{
		mutator: mutator,
		configs: configs,
		server:  &server,
	}

	kpr, err := newKeypairReloader(*mw.configs.CertFilePath, *mw.configs.KeyFilePath)
	if err != nil {
		return nil, err
	}

	mw.fileWatcher = kpr.fileWatcher

	if err := http2.ConfigureServer(mw.server, nil); err != nil {
		return nil, err
	}

	server.TLSConfig.GetCertificate = kpr.GetCertificateFunc()

	mux.HandleFunc("/", mw.handleRoot)
	mux.HandleFunc("/_healthz", mw.handleHealthz)
	mux.HandleFunc("/_ready", mw.handleHealthz)
	mux.HandleFunc("/mutate", mw.handleMutate)

	return mw, nil
}

// Starts the webserver and serves:
// - a welcome message on /
// - the passed Mutator on /mutate
// - a health probe on /_healthz
// - a readiness probe on /_ready
func (mw *mutatingWebhook) ListenAndServe() error {

	klog.Infof("Listening on %s\n", *mw.configs.Addr)

	ln, err := net.Listen("tcp", *mw.configs.Addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(ln, mw.server.TLSConfig)
	return mw.server.Serve(tlsListener)
}

// Shuts down the server and any resources it's using.
func (mw *mutatingWebhook) Shutdown(ctx context.Context) error {
	var errors *multierror.Error

	if err := mw.server.Shutdown(ctx); err != nil {
		errors = multierror.Append(errors, err)
	}

	if err := mw.fileWatcher.Close(); err != nil {
		errors = multierror.Append(errors, err)
	}

	return errors.ErrorOrNil()
}
