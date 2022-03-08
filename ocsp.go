package dynproxy

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

const ocspMime = "application/ocsp-request"

type ocspCacheEntity struct {
	SerialNumber                                  *big.Int
	Status                                        int
	RevocationReason                              int
	ProducedAt, ThisUpdate, NextUpdate, RevokedAt time.Time
	rawResp                                       []byte
}

type OCSPProcessor struct {
	ctx                    context.Context
	ocspStapleEnabled      bool
	ocspValidationEnabled  bool
	ocspCacheEnabled       bool
	ocspAutoRenewalEnabled bool
	ocspResponderUrl       string
	cache                  *ristretto.Cache
	events                 chan *Event
}

func NewOcspProcessor(ocspCtx context.Context, frConfig FrontendConfig, events chan *Event) *OCSPProcessor {
	ocspProcessor := &OCSPProcessor{
		ocspStapleEnabled:      frConfig.OcspStapleEnabled,
		ocspValidationEnabled:  frConfig.OcspValidationEnabled,
		ctx:                    ocspCtx,
		ocspResponderUrl:       frConfig.OcspResponderUrl,
		ocspCacheEnabled:       frConfig.OcspCacheEnabled,
		ocspAutoRenewalEnabled: frConfig.OcspAutoRenewalEnabled,
		events:                 events,
	}
	ocspProcessor.initCache()
	return ocspProcessor
}

func (p *OCSPProcessor) OcspVerify(cert, issuer *x509.Certificate) error {
	if p.ocspValidationEnabled {
		cacheEntity := p.getFromCache(cert)
		if cacheEntity != nil {
			return p.processOcspResponse(cacheEntity.SerialNumber, cacheEntity.Status, cert)
		}
		go p.backgroundOcspVerify(cert, issuer)
		return lazyLoadStaple
	}
	return nil
}

func (p *OCSPProcessor) backgroundOcspVerify(cert, issuer *x509.Certificate) {
	ocspResp, rawResp, err := p.ocspRequest(cert, issuer)
	if err != nil {
		p.events <- buildOcspErrorEvent(cert.SerialNumber.String(), UnavailableOcspResponderError, err, "ocsp request error.")
	} else {
		err = p.processOcspResponse(ocspResp.SerialNumber, ocspResp.Status, cert)
		if err != nil {
			p.events <- buildOcspErrorEvent(cert.SerialNumber.String(), OcspValidationError, err, "ocsp parse error, kill session")
		}
	}
	if p.ocspCacheEnabled {
		// TODO: Need to store error type
		p.storeToCache(rawResp, ocspResp)
	}
}

func (p *OCSPProcessor) GetOcspStaple(cert, issuer *x509.Certificate) ([]byte, error) {
	if p.ocspStapleEnabled {
		cacheEntity := p.getFromCache(cert)
		if cacheEntity != nil {
			return cacheEntity.rawResp, nil
		}
		ocspResp, rawResp, err := p.ocspRequest(cert, issuer)
		if err != nil {
			return nil, err
		}
		err = p.processOcspResponse(ocspResp.SerialNumber, ocspResp.Status, cert)
		if err != nil {
			return nil, err
		}
		if p.ocspCacheEnabled {
			p.storeToCache(rawResp, ocspResp)
		}
		return rawResp, nil
	}
	return nil, nil
}

func (p *OCSPProcessor) initCache() {
	if p.ocspCacheEnabled {
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: 100000,
			MaxCost:     10000,
			BufferItems: 128,
			Metrics:     false,
		})
		if err != nil {
			log.Warn().Msgf("can't init cache:%+v", err)
			return
		}
		p.cache = cache
	}
}

func (p *OCSPProcessor) getFromCache(cert *x509.Certificate) *ocspCacheEntity {
	if p.ocspCacheEnabled {
		ocspEntity, ok := p.cache.Get(cert.SerialNumber.String())
		if ok {
			cacheEntity, ok := ocspEntity.(*ocspCacheEntity)
			if ok {
				return cacheEntity
			}
		}
	}
	return nil
}

func (p *OCSPProcessor) ocspRequest(cert *x509.Certificate, issuer *x509.Certificate) (*ocsp.Response, []byte, error) {
	request, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{Hash: crypto.SHA256})
	if err != nil {
		return nil, nil, err
	}
	respBytes, err := p.sendOcspRequest(request)
	if err != nil {
		return nil, nil, err
	}
	ocspResp, err := ocsp.ParseResponse(respBytes, issuer)
	if err != nil {
		return nil, nil, err
	}
	return ocspResp, respBytes, nil
}

func (p *OCSPProcessor) sendOcspRequest(request []byte) ([]byte, error) {
	rsp, err := http.Post(p.ocspResponderUrl, ocspMime, bytes.NewReader(request))
	defer func(resp *http.Response) {
		if resp != nil {
			if resp.Body != nil {
				err := resp.Body.Close()
				if err != nil {
					log.Printf("got error while close http response: %+v", err)
				}
			}
		}
	}(rsp)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *OCSPProcessor) processOcspResponse(sn *big.Int, status int, cert *x509.Certificate) error {
	if sn.Cmp(cert.SerialNumber) != 0 {
		return incorrectSn
	}
	if status == ocsp.Revoked {
		return revokedCert
	}
	return nil
}

func (p *OCSPProcessor) storeToCache(rawResp []byte, resp *ocsp.Response) {
	if p.ocspCacheEnabled {
		entity := &ocspCacheEntity{
			SerialNumber:     resp.SerialNumber,
			Status:           resp.Status,
			RevocationReason: resp.RevocationReason,
			NextUpdate:       resp.NextUpdate,
			RevokedAt:        resp.RevokedAt,
			ProducedAt:       resp.ProducedAt,
			ThisUpdate:       resp.ThisUpdate,
			rawResp:          rawResp,
		}
		ttl := resp.NextUpdate.Sub(time.Now())
		p.cache.SetWithTTL(resp.SerialNumber.String(), entity, 1, ttl)
	}
}
