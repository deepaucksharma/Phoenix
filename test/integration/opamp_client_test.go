package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
)

// generateCerts creates a CA, server, and client certificate for mTLS testing.
func generateCerts() (caPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM []byte, err error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return
	}
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return
	}
	serverCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		return
	}
	clientCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	clientKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})
	return
}

// mockOpAMPServer simulates minimal OpAMP interactions over HTTPS.
type mockOpAMPServer struct {
	policy []byte
	patch  interfaces.ConfigPatch
	mu     sync.Mutex
	status int
}

func (s *mockOpAMPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/policy":
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(s.policy)
	case "/patch":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.patch)
	case "/status":
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		s.mu.Lock()
		s.status++
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

func (s *mockOpAMPServer) StatusCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func TestOpAMPClientIntegration(t *testing.T) {
	ca, srvCert, srvKey, cliCert, cliKey, err := generateCerts()
	require.NoError(t, err)

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(ca)

	serverTLS, err := tls.X509KeyPair(srvCert, srvKey)
	require.NoError(t, err)

	server := &mockOpAMPServer{}
	ts := httptest.NewUnstartedServer(server)
	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverTLS},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	ts.StartTLS()
	defer ts.Close()

	// Write client certs to temp files
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	os.WriteFile(caFile, ca, 0644)
	certFile := filepath.Join(dir, "client.pem")
	os.WriteFile(certFile, cliCert, 0644)
	keyFile := filepath.Join(dir, "client.key")
	os.WriteFile(keyFile, cliKey, 0644)

	// Prepare policy YAML
	server.policy = []byte(`global_settings:
  autonomy_level: shadow
  collector_cpu_safety_limit_mcores: 200
  collector_rss_safety_limit_mib: 200

processors_config:
  adaptive_topk:
    enabled: true
    k_value: 30
    k_min: 10
    k_max: 60

pic_control_config:
  policy_file_path: ""
  max_patches_per_minute: 10
  patch_cooldown_seconds: 1
`)

	server.patch = interfaces.ConfigPatch{
		PatchID:             "patch1",
		TargetProcessorName: component.MustNewID("adaptive_topk"),
		ParameterPath:       "k_value",
		NewValue:            40,
		Reason:              "test",
		Severity:            "normal",
		Source:              "opamp",
		Timestamp:           time.Now().Unix(),
	}

	// Build extension
	host := &componenttest.TestHost{}
	factory := pic_control_ext.NewFactory()
	cfg := factory.CreateDefaultConfig().(*pic_control_ext.Config)
	cfg.PolicyFilePath = ""
	cfg.OpAMPConfig = &pic_control_ext.OpAMPClientConfig{
		ServerURL:           ts.URL,
		InsecureSkipVerify:  false,
		ClientCertFile:      certFile,
		ClientKeyFile:       keyFile,
		CACertFile:          caFile,
		PollIntervalSeconds: 1,
	}

	ext, err := factory.CreateExtension(context.Background(), component.ExtensionCreateSettings{Logger: zap.NewNop()}, cfg)
	require.NoError(t, err)

	// Create processor
	procFactory := adaptive_topk.NewFactory()
	procCfg := procFactory.CreateDefaultConfig().(*adaptive_topk.Config)
	procCfg.KValue = 30
	procCfg.KMin = 10
	procCfg.KMax = 60
	procCfg.ResourceField = "process.name"
	procCfg.CounterField = "process.cpu_seconds_total"
	sink := new(consumertest.MetricsSink)
	proc, err := procFactory.CreateMetricsProcessor(context.Background(), processor.CreateSettings{}, procCfg, sink)
	require.NoError(t, err)

	host.AddExtension(component.MustNewID("pic_control"), ext)
	host.AddProcessor(component.MustNewID("adaptive_topk"), proc)

	require.NoError(t, ext.Start(context.Background(), host))
	require.NoError(t, proc.Start(context.Background(), host))

	// Wait for polling
	time.Sleep(2 * time.Second)

	status, err := proc.(interfaces.UpdateableProcessor).GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 40, status.Parameters["k_value"])
	assert.Greater(t, server.StatusCount(), 0)

	ext.Shutdown(context.Background())
	proc.Shutdown(context.Background())
}
