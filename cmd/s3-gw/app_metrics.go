package main

import (
	"net/http"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewPrometheusService creates a new service for gathering prometheus metrics.
func NewPrometheusService(v *viper.Viper, log *zap.Logger, handler http.Handler) *Service {
	if log == nil {
		return nil
	}

	return &Service{
		Server: &http.Server{
			Addr:    v.GetString(cfgPrometheusAddress),
			Handler: handler,
		},
		enabled:     v.GetBool(cfgPrometheusEnabled),
		serviceType: "Prometheus",
		log:         log.With(zap.String("service", "Prometheus")),
	}
}
