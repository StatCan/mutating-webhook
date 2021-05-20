package mutatingwebhook

import (
	"crypto/tls"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
)

// Based on:
// https://github.com/robustirc/bridge/blob/6b0e8ab3736b2b847b70a924971df0d0024d6b79/tlsutil/tlsutil.go
// https://stackoverflow.com/questions/37473201/is-there-a-way-to-update-the-tls-certificates-in-a-net-http-server-without-any-d/40883377#40883377

type keypairReloader struct {
	certMu      sync.RWMutex
	cert        *tls.Certificate
	fileWatcher *fsnotify.Watcher
	certPath    string
	keyPath     string
}

// Creates the struct that allows for the management of the certificate reloading.
func newKeypairReloader(certPath, keyPath string) (*keypairReloader, error) {
	result := &keypairReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	result.cert = &cert

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	result.fileWatcher = watcher
	err = result.fileWatcher.Add(certPath)
	if err != nil {
		klog.Fatal(err)
	}
	err = result.fileWatcher.Add(keyPath)
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					klog.Infof("TLS Cert or Key updated - reloading")
					if err := result.maybeReload(); err != nil {
						klog.Errorf("Could not reload: %v", err)
					} else {
						klog.Infof("Reload complete")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				klog.Error(err)
			}
		}
	}()

	return result, nil
}

// Attempts to reload the certificates.
func (kpr *keypairReloader) maybeReload() error {
	newCert, err := tls.LoadX509KeyPair(kpr.certPath, kpr.keyPath)
	if err != nil {
		return err
	}
	kpr.certMu.Lock()
	defer kpr.certMu.Unlock()
	kpr.cert = &newCert
	return nil
}

// Function which is used to replace
// http.Server.TLSConfig.GetCertificate so that the certificates can be reloaded.
func (kpr *keypairReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		kpr.certMu.RLock()
		defer kpr.certMu.RUnlock()
		return kpr.cert, nil
	}
}
