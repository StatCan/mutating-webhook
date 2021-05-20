package mutatingwebhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/http2"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
)

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
	ListenAndMutate()
	Shutdown(ctx context.Context)
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

	// Make sure content type is correct
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	// Decode the request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	// Attempt to get the AdmissionReview the request
	admissionReview := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	// Evaluate/Mutate the AdmissionRequest.
	response, err := mw.mutator.Mutate(*admissionReview.Request)
	if err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	reviewResponse := v1.AdmissionReview{
		Response: &response,
	}

	if body, err = json.Marshal(reviewResponse); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

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
) MutatingWebhook {

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
		klog.Fatal(err)
		// return err
	}
	mw.fileWatcher = kpr.fileWatcher

	if err := http2.ConfigureServer(mw.server, nil); err != nil {
		klog.Fatal(err)
	}

	server.TLSConfig.GetCertificate = kpr.GetCertificateFunc()

	mux.HandleFunc("/", mw.handleRoot)
	mux.HandleFunc("/_healthz", mw.handleHealthz)
	mux.HandleFunc("/mutate", mw.handleMutate)

	return mw
}

// Starts the webserver and serves:
// - a welcome message on /
// - the passed Mutator on /mutate
// - a health probe on /_healthz
func (mw *mutatingWebhook) ListenAndMutate() {

	klog.Infof("Listening on %s\n", *mw.configs.Addr)

	ln, err := net.Listen("tcp", *mw.configs.Addr)
	if err != nil {
		klog.Fatal(err)
	}

	tlsListener := tls.NewListener(ln, mw.server.TLSConfig)
	mw.server.Serve(tlsListener) //, *mw.configs.CertFilePath, *mw.configs.KeyFilePath))
}

// Shuts down the server and any resources it's using.
func (mw *mutatingWebhook) Shutdown(ctx context.Context) {
	mw.server.Shutdown(ctx)
	err := mw.fileWatcher.Close()
	if err != nil {
		klog.Error(err)
	}
}
