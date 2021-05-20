package mutatingwebhook

import "time"

// Default values used to fill the MutatingWebhookConfigs
var (
	addr           = ":8443"
	readTimeout    = 10 * time.Second
	writeTimeout   = 10 * time.Second
	maxHeaderBytes = 0
	certFilePath   = "./certs/tls.crt"
	keyFilePath    = "./certs/tls.key"
)

// Any values left nil will use default values.
type MutatingWebhookConfigs struct {
	// Addr optionally specifies the TCP address for the server to listen on,
	// in the form "host:port". If empty, ":8443" is used.
	// The service names are defined in RFC 6335 and assigned by IANA.
	// See net.Dial for details of the address format.
	Addr           *string
	ReadTimeout    *time.Duration
	WriteTimeout   *time.Duration
	MaxHeaderBytes *int // When 0, defaults to the maximum
	// The file path to the certificate file.
	CertFilePath *string
	// The file path to the key file from which the certificate is derived.
	KeyFilePath *string
}

// Sets default values.
// This allows for simpler use of the library.s
func setDefaults(configs MutatingWebhookConfigs) MutatingWebhookConfigs {
	if configs.Addr == nil {
		configs.Addr = &addr
	}

	if configs.ReadTimeout == nil {
		configs.ReadTimeout = &readTimeout
	}

	if configs.WriteTimeout == nil {
		configs.WriteTimeout = &writeTimeout
	}

	if configs.MaxHeaderBytes == nil {
		configs.MaxHeaderBytes = &maxHeaderBytes
	}

	if configs.CertFilePath == nil {
		configs.CertFilePath = &certFilePath
	}

	if configs.KeyFilePath == nil {
		configs.KeyFilePath = &keyFilePath
	}

	return configs
}
