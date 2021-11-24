package dynproxy

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"golang.org/x/crypto/ocsp"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

const ocspMime = "application/ocsp-request"

type OCSPProcessor struct {
	ctx                    context.Context
	ocspStapleEnabled      bool
	ocspResponderUrl       string
	ocspCacheEnabled       bool
	ocspAutoRenewalEnabled bool
}

func (o *OCSPProcessor) OcspVerify(cert, issuer *x509.Certificate) ([]byte, error) {
	request, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{Hash: crypto.SHA256})
	if err != nil {
		return nil, err
	}
	response, err := o.sendOcspRequest(request)
	if err != nil {
		return nil, err
	}
	ocspResp, err := ocsp.ParseResponse(response, issuer)
	if err != nil {
		return nil, err
	}
	o.processOcspResponse(ocspResp)
	return response, nil
}

func (o *OCSPProcessor) sendOcspRequest(request []byte) ([]byte, error) {
	rsp, err := http.Post(o.ocspResponderUrl, ocspMime, bytes.NewReader(request))
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("got error while close http response: %+v", err)
		}
	}(rsp.Body)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// TODO: for the further implementation
func (o *OCSPProcessor) processOcspResponse(resp *ocsp.Response) {

}
