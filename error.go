package dynproxy

import "errors"

var noActiveBackends = errors.New("no active backends")
var balancerNotFound = errors.New("invalid balancer name")
var noSessionFound = errors.New("no session found")
var closedSession = errors.New("closed session")
