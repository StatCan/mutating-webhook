package mutatingwebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDefault(t *testing.T) {
	configs := MutatingWebhookConfigs{}

	configs = setDefaults(configs)

	assert.Equal(t, *configs.Addr, addr)
	assert.Equal(t, *configs.ReadTimeout, readTimeout)
	assert.Equal(t, *configs.WriteTimeout, writeTimeout)
	assert.Equal(t, *configs.MaxHeaderBytes, maxHeaderBytes)
	assert.Equal(t, *configs.CertFilePath, certFilePath)
	assert.Equal(t, *configs.KeyFilePath, keyFilePath)
}
