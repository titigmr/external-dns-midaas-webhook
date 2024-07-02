package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"github.com/titigmr/external-dns-midaas-wehbook/api"
	"github.com/titigmr/external-dns-midaas-wehbook/midaas"
	"sigs.k8s.io/external-dns/endpoint"
)

type Config struct {
	Server struct {
		Port string `envconfig:"API_SERVER_PORT" default:"6666"`
		Host string `envconfig:"API_SERVER_HOST" default:"127.0.0.1"`
	}
	Options struct {
		ReadTimeout  time.Duration `envconfig:"API_READ_TIMEOUT"   default:"3s"`
		WriteTimeout time.Duration `envconfig:"API_WRITE_TIMEOUT"  default:"3s"`
	}
	Provider struct {
		SkipTlsVerify bool   `envconfig:"PROVIDER_SKIP_TLS_VERIFY"  default:"true"`
		ZoneSuffix    string `envconfig:"PROVIDER_DNS_ZONE_SUFFIX"  default:"dev.example.com"`
		WsUrl         string `envconfig:"PROVIDER_WS_URL"           default:"https://example.com/ws"`
	}
}

func main() {

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableColors: false,
	})

	var cfg Config

	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	domainFilters := endpoint.DomainFilter{}
	tsigs := make([]midaas.TSIGCredentials, 1)
	tsigs[1] = midaas.TSIGCredentials{Keyname: "ddns-key.d301", Keyvalue: "jdzudhzuZD=="}
	p, err := midaas.NewMiDaasProvider(cfg.Provider.WsUrl, tsigs, domainFilters, cfg.Provider.ZoneSuffix, cfg.Provider.SkipTlsVerify)

	if err != nil {
		log.Fatal(err)
	}
	api.StartHTTPApi(p, cfg.Options.ReadTimeout, cfg.Options.WriteTimeout, fmt.Sprintf("%v:%v", cfg.Server.Host, cfg.Server.Port))
}
