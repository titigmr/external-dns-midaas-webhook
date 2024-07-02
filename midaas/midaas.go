package midaas

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

const (
	dnsCreate  = "CREATE"
	dnsDelete  = "DELETE"
	dnsUpdate  = "UPDATE"
	defaultTTL = 3600
)

type TSIGCredentials struct {
	Keyname  string
	Keyvalue string
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Body struct {
	TTL      int    `json:"ttl,omitempty"`
	Keyname  string `json:"keyname"`
	Keyvalue string `json:"keyvalue"`
}

type ZoneRecord struct {
	DNSName string
	Target  string
	Type    string
	TTL     int
	TSIG    TSIGCredentials
}

type ChangeRecord struct {
	Action     string
	ZoneRecord ZoneRecord
}

type midaasProvider struct {
	provider.BaseProvider
	tsig         []TSIGCredentials
	domainFilter endpoint.DomainFilter
	wsUrl        string
}

func NewMiDaasProvider(WsUrl string, tsig []TSIGCredentials, domainFilter endpoint.DomainFilter, dnsSuffix string, skipTlsVerify bool) (provider.Provider, error) {
	if len(tsig) == 0 {
		return nil, fmt.Errorf("no midaas tsig provided, use TSIG_ZONE_<Zone>=<Keyvalue> env vars ")
	}

	UpdateHttpClient(skipTlsVerify)
	GetAvailableZones(dnsSuffix, &domainFilter, &tsig)

	provider := &midaasProvider{
		tsig:         tsig,
		wsUrl:        WsUrl,
		domainFilter: domainFilter,
	}
	return provider, nil
}

func GetAvailableZones(dnsSuffix string, domainFilter *endpoint.DomainFilter, tsig *[]TSIGCredentials) *endpoint.DomainFilter {
	filters := &domainFilter.Filters
	for _, config := range *tsig {
		ss := strings.Split(config.Keyname, ".")
		zone := fmt.Sprintf("%v.%v", ss[len(ss)-1], dnsSuffix)
		log.WithField("zone", zone).Info("Successfully parsed zones")
		*filters = append(*filters, zone)
	}
	return domainFilter
}

func GetTSIGFromTarget(tsigs *[]TSIGCredentials, zone string) *TSIGCredentials {
	for _, tsig := range *tsigs {
		parsedZone := strings.Split(tsig.Keyname, ".")[1]
		if strings.Contains(zone, parsedZone) {
			return &tsig
		}
	}
	return &TSIGCredentials{}
}

func (p *midaasProvider) MatchTargetInZone(target string) *TSIGCredentials {
	for _, domain := range p.domainFilter.Filters {
		log.WithFields(log.Fields{
			"target": target,
			"domain": domain}).Debug("Check target for domain")
		if strings.HasSuffix(target, domain) {
			allowedSig := GetTSIGFromTarget(&p.tsig, domain)
			log.WithFields(log.Fields{
				"target": target,
				"domain": domain}).Debug("TSIG credentials found for target")
			return allowedSig
		}
	}
	log.Errorf("Target %v not allowed for any domain", target)
	return &TSIGCredentials{}
}

func (b *Body) UpdateTTL(ttl int) { b.TTL = ttl }

func (p *midaasProvider) ApplyRecord(action string, recordInfo *ZoneRecord) error {
	if recordInfo.TSIG == (TSIGCredentials{}) {
		return fmt.Errorf("unavalaible TSIG to %v %v", action, recordInfo.DNSName)
	}

	url := fmt.Sprintf("%v/%v/%v/%v", p.wsUrl, recordInfo.DNSName, recordInfo.Type, recordInfo.Target)
	b := Body{Keyname: recordInfo.TSIG.Keyname, Keyvalue: recordInfo.TSIG.Keyvalue}

	switch action {
	case dnsUpdate, dnsCreate:
		b.UpdateTTL(recordInfo.TTL)
		err := RequestUrl("PUT", url, b)
		if err != nil {
			return fmt.Errorf("%v failed: %v", action, err)
		}
	case dnsDelete:
		err := RequestUrl("DELETE", url, b)
		if err != nil {
			return fmt.Errorf("%v failed: %v", action, err)
		}
	}
	log.WithFields(log.Fields{"DNSName": recordInfo.DNSName,
		"Targets":    recordInfo.Target,
		"TTL":        recordInfo.TTL,
		"RecordType": recordInfo.Type}).Infof("Apply %v record", action)
	return nil
}

func (p *midaasProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	combinedChanges := make([]*ChangeRecord, 0, len(changes.Create)+len(changes.UpdateNew)+len(changes.Delete))

	combinedChanges = append(combinedChanges, p.newChanges(dnsCreate, changes.Create)...)
	combinedChanges = append(combinedChanges, p.newChanges(dnsUpdate, changes.UpdateNew)...)
	combinedChanges = append(combinedChanges, p.newChanges(dnsDelete, changes.Delete)...)

	errorList := []error{}

	for _, action := range combinedChanges {
		err := p.ApplyRecord(action.Action, &action.ZoneRecord)
		if err != nil {
			errorList = append(errorList, err)
			log.Error(err)
		}
	}
	if len(errorList) > 0 {
		return fmt.Errorf("%v error(s) for applying changes", len(errorList))
	}

	return nil
}

func newChange(action string, e *endpoint.Endpoint, tsig TSIGCredentials) *ChangeRecord {
	ttl := defaultTTL
	if e.RecordTTL.IsConfigured() {
		ttl = int(e.RecordTTL)
	}

	change := &ChangeRecord{
		Action: action,
		ZoneRecord: ZoneRecord{
			DNSName: e.DNSName,
			Type:    e.RecordType,
			Target:  e.Targets[0],
			TTL:     ttl,
			TSIG:    tsig,
		},
	}
	return change
}

func (p *midaasProvider) newChanges(action string, endpoints []*endpoint.Endpoint) []*ChangeRecord {
	changes := make([]*ChangeRecord, 0, len(endpoints))
	for _, e := range endpoints {
		tsig := p.MatchTargetInZone(e.DNSName)
		changes = append(changes, newChange(action, e, *tsig))
	}
	return changes
}

func RequestUrl(method string, url string, body Body) error {
	client := &http.Client{}
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	request.Header = http.Header{"Content-Type": {"application/json"}}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	bodyResponse, err := io.ReadAll(response.Body)
	defer response.Body.Close()

	if response.StatusCode > 200 {
		if err != nil {
			return err
		}
		return fmt.Errorf(string(bodyResponse))
	}

	var respFormat Response
	respBody := &respFormat
	json.Unmarshal(bodyResponse, respBody)
	log.WithFields(log.Fields{"message": respBody.Message,
		"status": respBody.Status, "url": url}).Debug("DNS api reponses")

	if respBody.Status != "OK" {
		return fmt.Errorf(respBody.Message)
	}
	return nil
}

func GetUrl(url string) (*http.Response, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode > 200 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(string(body))
	}
	return response, nil
}

func (p *midaasProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	endpoints := []*endpoint.Endpoint{}

	// get all domains
	for _, zone := range p.domainFilter.Filters {
		var url string = fmt.Sprintf("%v/%v", p.wsUrl, zone)
		resp, err := GetUrl(url)
		if err != nil {
			return nil, err
		}

		// parse zone records
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()

		if err != nil {
			return nil, err
		}
		// struct is not predictable
		var zoneTargets map[string]interface{}
		json.Unmarshal([]byte(string(body)), &zoneTargets)

		// for each index get target, ttl, type record and dnsName
		for index := range zoneTargets {
			v := zoneTargets[index].(map[string]interface{})
			intTTL, err := strconv.ParseInt(fmt.Sprint(v["ttl"]), 10, 64)
			if err != nil {
				return nil, err
			}

			// Create Endpoint object
			e := endpoint.Endpoint{DNSName: strings.Split(index, "./")[0],
				Targets:    endpoint.Targets([]string{fmt.Sprint(v["value"])}),
				RecordTTL:  endpoint.TTL(intTTL),
				RecordType: fmt.Sprint(v["type"])}

			log.WithFields(log.Fields{"DNSName": e.DNSName,
				"Targets":    e.Targets,
				"TTL":        e.RecordTTL,
				"RecordType": e.RecordType}).Debug("Returned endpoint")
			// add parsed Endpoint into slice
			endpoints = append(endpoints, &e)
		}
	}
	return endpoints, nil
}

func UpdateHttpClient(skipTlsVerify bool) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: skipTlsVerify}
}

func (p *midaasProvider) GetDomainFilter() endpoint.DomainFilter {
	return p.domainFilter
}

func (p midaasProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	return endpoints, nil
}
