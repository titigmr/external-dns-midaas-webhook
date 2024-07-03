package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Showmax/env"
	"github.com/kelseyhightower/envconfig"
	"github.com/titigmr/external-dns-midaas-webhook/api"
	"github.com/titigmr/external-dns-midaas-webhook/midaas"
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
		LogLevel     string        `envconfig:"API_LOG_LEVEL"      default:"INFO"`
	}
	Provider struct {
		SkipTlsVerify bool   `envconfig:"PROVIDER_SKIP_TLS_VERIFY"  default:"true"`
		ZoneSuffix    string `envconfig:"PROVIDER_DNS_ZONE_SUFFIX"  default:"dev.example.com"`
		WsUrl         string `envconfig:"PROVIDER_WS_URL"           default:"https://example.com/ws"`
	}
}

type TSIGConfig struct {
	Map map[string]string `env:"ZONE_"`
}

func main() {

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	level, err := log.ParseLevel(cfg.Options.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// parse zone config
	domainFilters := endpoint.DomainFilter{}
	var tsigCfg TSIGConfig
	errParsing := env.Load(&tsigCfg, "TSIG_")
	if errParsing != nil {
		log.Fatal(errParsing)
	}

	var tsigs []midaas.TSIGCredentials
	for key, value := range tsigCfg.Map {
		tsigs = append(tsigs, midaas.TSIGCredentials{Keyname: fmt.Sprintf("%v%v", "ddns-key.", key), Keyvalue: value})
	}

	// create provider
	p, err := midaas.NewMiDaasProvider(strings.TrimRight(cfg.Provider.WsUrl,"/"), tsigs, domainFilters, cfg.Provider.ZoneSuffix, cfg.Provider.SkipTlsVerify)

	if err != nil {
		log.Fatal(err)
	}
	api.StartHTTPApi(p, cfg.Options.ReadTimeout, cfg.Options.WriteTimeout, fmt.Sprintf("%v:%v", cfg.Server.Host, cfg.Server.Port))
}
