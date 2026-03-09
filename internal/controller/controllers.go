package controllers

import (
	"github.com/netobserv/netobserv-operator/internal/controller/flp"
	"github.com/netobserv/netobserv-operator/internal/controller/monitoring"
	"github.com/netobserv/netobserv-operator/internal/controller/networkpolicy"
	"github.com/netobserv/netobserv-operator/internal/controller/static"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
)

var Registerers = []manager.Registerer{Start, flp.Start, monitoring.Start, networkpolicy.Start, static.Start}
