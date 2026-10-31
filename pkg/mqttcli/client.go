package mqttcli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type Config struct {
	Broker   string
	CAFile   string
	SNI      string
	User     string
	Pass     string
	ClientID string
	Will     *Message
}

type Message struct {
	Topic   string
	Payload []byte
	Retain  bool
}

type MessageHandler func(topic string, payload []byte)

type Client struct {
	cli     mqtt.Client
	log     *zap.Logger
	filters []string
}

func New(cfg Config, log *zap.Logger, handler MessageHandler, filters []string, onConnect func(*Client)) (*Client, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("mqttcli: client id required")
	}

	c := &Client{log: log, filters: filters}

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.User).
		SetPassword(cfg.Pass).
		SetKeepAlive(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetOrderMatters(false)

	if cfg.Will != nil {
		opts.SetWill(cfg.Will.Topic, string(cfg.Will.Payload), 1, cfg.Will.Retain)
	}

	if cfg.CAFile != "" {
		tlsConf, err := buildTLSConfig(cfg.CAFile, cfg.SNI)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConf)
	}

	opts.SetOnConnectHandler(func(cl mqtt.Client) {
		log.Info("mqtt connected", zap.String("broker", cfg.Broker))
		for _, filter := range filters {
			tok := cl.Subscribe(filter, 1, func(_ mqtt.Client, m mqtt.Message) {
				handler(m.Topic(), m.Payload())
			})
			go func(filter string, tok mqtt.Token) {
				if !tok.WaitTimeout(5 * time.Second) {
					log.Warn("mqtt subscribe timeout", zap.String("filter", filter))
					return
				}
				if err := tok.Error(); err != nil {
					log.Warn("mqtt subscribe failed", zap.String("filter", filter), zap.Error(err))
				}
			}(filter, tok)
		}
		if onConnect != nil {
			onConnect(c)
		}
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Warn("mqtt connection lost", zap.Error(err))
	})

	c.cli = mqtt.NewClient(opts)
	return c, nil
}

func (c *Client) Connect() error {
	tok := c.cli.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		return errors.New("mqttcli: connect timeout")
	}
	return tok.Error()
}

func (c *Client) Publish(topic string, payload []byte) error {
	return c.publish(topic, payload, false)
}

func (c *Client) PublishRetained(topic string, payload []byte) error {
	return c.publish(topic, payload, true)
}

func (c *Client) publish(topic string, payload []byte, retain bool) error {
	if !c.cli.IsConnected() {
		return errors.New("mqttcli: not connected")
	}
	tok := c.cli.Publish(topic, 1, retain, payload)
	if !tok.WaitTimeout(5 * time.Second) {
		return errors.New("mqttcli: publish timeout")
	}
	return tok.Error()
}

func (c *Client) IsConnected() bool {
	return c.cli.IsConnected()
}

func (c *Client) Disconnect() {
	c.cli.Disconnect(200)
}

func buildTLSConfig(caFile, sni string) (*tls.Config, error) {
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("mqttcli: read ca: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("mqttcli: no certificate parsed from %s", caFile)
	}

	conf := &tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}
	if sni != "" {
		conf.ServerName = sni
		return conf, nil
	}

	// 自签证书可能没有 SAN，未配置 SNI 时退化为只校验证书链，不校验主机名。
	conf.InsecureSkipVerify = true
	conf.VerifyPeerCertificate = func(raw [][]byte, _ [][]*x509.Certificate) error {
		if len(raw) == 0 {
			return errors.New("mqttcli: no peer certificate")
		}
		certs := make([]*x509.Certificate, 0, len(raw))
		for _, b := range raw {
			cert, err := x509.ParseCertificate(b)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		inter := x509.NewCertPool()
		for _, cert := range certs[1:] {
			inter.AddCert(cert)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{Roots: roots, Intermediates: inter})
		return err
	}
	return conf, nil
}
