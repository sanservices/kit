package tls

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

// TLS is the struct representation of the tls configuration
type TLS struct {
	// CACertPEM is the path to the CA certificate
	CACertPEM string `yaml:"ca_cert_pem"`
	// CertPEM is the path to the client certificate
	CertPEM string `yaml:"cert_pem"`
	// KeyPEM is the path to the client key
	KeyPEM string `yaml:"key_pem"`
	// SkipVerify is a flag to skip verification of the certificate
	SkipVerify bool `yaml:"skip_verify"`
	// Timeout is the timeout for the http request
	TimeoutSecs int `yaml:"timeout_secs"`
}

func GetTLSConf(conf TLS) (*tls.Config, error) {

	caCertPEM, err := ioutil.ReadFile(conf.CACertPEM)
	if err != nil {
		log.Println("Could not open caCertPem", err)
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCertPEM)

	cerPEM, err := tls.LoadX509KeyPair(conf.CertPEM, conf.KeyPEM)
	if err != nil {
		log.Println("Could not open certPem", err)
		return nil, err
	}

	t := tls.Config{
		Certificates:       []tls.Certificate{cerPEM},
		InsecureSkipVerify: conf.SkipVerify,
		RootCAs:            certPool,
	}

	t.BuildNameToCertificate()

	return &t, err
}

func GetHTTPSClient(tlsConf *tls.Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
			TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			Dial: func(network string, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, 10*time.Second)
			},
		},
	}
}
