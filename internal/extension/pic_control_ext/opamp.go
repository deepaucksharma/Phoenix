package pic_control_ext

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
	"net/http"
)

// startOpAMPClient starts the OpAMP client
func (e *Extension) startOpAMPClient(ctx context.Context) error {
	cfg := e.config.OpAMPConfig
	if cfg == nil {
		return nil
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.InsecureSkipVerify}

	if cfg.CACertFile != "" {
		caData, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		tlsCfg.RootCAs = pool
	}

	if cfg.ClientCertFile != "" && cfg.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.pollOpAMPServer(ctx, client)
			}
		}
	}()

	return nil
}