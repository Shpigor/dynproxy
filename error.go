package dynproxy

import "errors"

var noActiveBackends = errors.New("no active backends")
var balancerNotFound = errors.New("invalid balancer name")
