package dynproxy

import "errors"

var noActiveBackends = errors.New("no active backends")
var balancerNotFound = errors.New("invalid balancer name")
var noSessionFound = errors.New("no session found")
var closedSession = errors.New("closed session")

var revokedCert = errors.New("certificate is revoked")
var incorrectSn = errors.New("incorrect serial number")
var lazyLoadStaple = errors.New("lazy load of ocsp staple")
