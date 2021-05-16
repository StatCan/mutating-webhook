package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	v1 "k8s.io/api/admission/v1"
)

// Based on:
// https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec
// https://github.com/alex-leonhardt/k8s-mutate-webhook

// The Mutator interface is what is implemented to
// pass the mutation logic to the webserver.
type Mutator interface {
	Mutate(request v1.AdmissionRequest) (v1.AdmissionResponse, error)
}

func (mw *mutatingWebhook) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from mutating-webhook! Mutation available on: /mutate")
}

func (mw *mutatingWebhook) handleHealthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

// handleMutate is what wraps the Mutator and serves the logic
func (mw *mutatingWebhook) handleMutate(w http.ResponseWriter, r *http.Request) {
	// Decode the request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	admissionReview := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	response, err := mw.mutator.Mutate(*admissionReview.Request)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	reviewResponse := v1.AdmissionReview{
		Response: &response,
	}

	if body, err = json.Marshal(reviewResponse); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

type mutatingWebhook struct {
	mutator Mutator
}

// Starts the webserver and serves:
// - a welcome message on /
// - the passed Mutator on /mutate
// - a health probe on /_healthz
func ListenAndMutate(
	mutator Mutator,
	configs MutatingWebhookConfigs,
) {
	configs = setDefaults(configs)

	mw := mutatingWebhook{
		mutator: mutator,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", mw.handleRoot)
	mux.HandleFunc("/_healthz", mw.handleHealthz)
	mux.HandleFunc("/mutate", mw.handleMutate)

	s := &http.Server{
		Addr:           *configs.Addr,
		Handler:        mux,
		ReadTimeout:    *configs.ReadTimeout,
		WriteTimeout:   *configs.WriteTimeout,
		MaxHeaderBytes: *configs.MaxHeaderBytes,
	}

	log.Printf("Listening on %s\n", *configs.Addr)
	log.Fatal(s.ListenAndServeTLS(*configs.CertFilePath, *configs.KeyFilePath))
}
