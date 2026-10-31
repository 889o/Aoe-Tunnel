package tunnel

import (
	stdtls "crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	tls "github.com/refraction-networking/utls"
)

// Certs 同时用于 QUIC 主隧道与 TLS 降级链路的 mTLS 认证。
// utls 服务端对 RequireAndVerifyClientCert 支持不完整，这里用 RequireAnyClientCert
// 加 VerifyPeerCertificate 手工校验，保持与 crypto/tls 等价的链校验语义。
type Certs struct {
	serverCert    tls.Certificate
	clientCert    tls.Certificate
	serverCertStd stdtls.Certificate
	clientCertStd stdtls.Certificate
	pool          *x509.CertPool
}

func LoadCerts(serverCertFile, serverKeyFile, clientCertFile, clientKeyFile, caFile string) (*Certs, error) {
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server key pair: %w", err)
	}
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client key pair: %w", err)
	}
	// 降级链路服务端用 crypto/tls，需要独立加载一份标准库证书。
	serverCertStd, err := stdtls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load std server key pair: %w", err)
	}
	clientCertStd, err := stdtls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load std client key pair: %w", err)
	}
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("ca file %s contains no certificate", caFile)
	}
	return &Certs{
		serverCert: serverCert, clientCert: clientCert,
		serverCertStd: serverCertStd, clientCertStd: clientCertStd,
		pool: pool,
	}, nil
}

func (c *Certs) ClientConfig(serverName, alpn string) *tls.Config {
	cert := c.clientCert
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      c.pool,
		ServerName:   serverName,
		NextProtos:   []string{alpn},
		MinVersion:   tls.VersionTLS13,
		// uTLS 自定义 ClientHello 时显式回调，确保服务端 CertificateRequest 能拿到客户端证书。
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}
}

func (c *Certs) ServerConfig(alpn string) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{c.serverCert},
		ClientAuth:   tls.RequireAnyClientCert,
		NextProtos:   []string{alpn},
		MinVersion:   tls.VersionTLS13,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return c.verifyClientChain(rawCerts)
		},
	}
}

// FallbackClientConfig 用于降级链路的 utls 客户端，ALPN 由 ClientHelloSpec 决定。
func (c *Certs) FallbackClientConfig(serverName string) *tls.Config {
	cert := c.clientCert
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      c.pool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS13,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}
}

// StdClientConfig 供反向隧道出口使用 crypto/tls 拨号入口的 TLS 监听。
func (c *Certs) StdClientConfig(serverName, alpn string) *stdtls.Config {
	return &stdtls.Config{
		Certificates: []stdtls.Certificate{c.clientCertStd},
		RootCAs:      c.pool,
		ServerName:   serverName,
		NextProtos:   []string{alpn},
		MinVersion:   stdtls.VersionTLS13,
	}
}

// StdServerConfig 供 crypto/tls 服务端使用，utls 服务端的客户端证书校验模式不完整。
func (c *Certs) StdServerConfig(alpn string) *stdtls.Config {
	return &stdtls.Config{
		Certificates: []stdtls.Certificate{c.serverCertStd},
		ClientAuth:   stdtls.RequireAnyClientCert,
		NextProtos:   []string{alpn},
		MinVersion:   stdtls.VersionTLS13,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return c.verifyClientChain(rawCerts)
		},
	}
}

func (c *Certs) verifyClientChain(rawCerts [][]byte) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("mtls: client certificate missing")
	}
	certs := make([]*x509.Certificate, 0, len(rawCerts))
	for _, raw := range rawCerts {
		cert, err := x509.ParseCertificate(raw)
		if err != nil {
			return fmt.Errorf("mtls: parse client cert: %w", err)
		}
		certs = append(certs, cert)
	}
	inter := x509.NewCertPool()
	for _, cert := range certs[1:] {
		inter.AddCert(cert)
	}
	_, err := certs[0].Verify(x509.VerifyOptions{
		Roots:         c.pool,
		Intermediates: inter,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return fmt.Errorf("mtls: verify client cert: %w", err)
	}
	return nil
}
