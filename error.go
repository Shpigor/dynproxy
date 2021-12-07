package dynproxy

import "errors"

var noActiveBackends = errors.New("no active backends")
var balancerNotFound = errors.New("invalid balancer name")
var noStreamFound = errors.New("no stream found")
var closedStream = errors.New("closed stream")
