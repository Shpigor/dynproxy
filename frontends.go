package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
)

func InitFrontends(ctx context.Context, config Config, events chan *Event) {
	for _, frConfig := range config.Frontends {
		frCtx := context.WithValue(ctx, "name", frConfig.Name)
		frontend := Frontend{
			Context:         frCtx,
			Net:             frConfig.Net,
			Address:         frConfig.Address,
			Name:            frConfig.Name,
			connChannel:     events,
			defaultBalancer: frConfig.BackendGroup,
			ocspProc:        NewOcspProcessor(context.WithValue(ctx, "name", "OCSP"), frConfig, events),
			TlsConfig: &TlsConfig{
				SkipVerify: frConfig.TlsSkipVerify,
				CACertPath: frConfig.TlsCACertPath,
				CertPath:   frConfig.TlsCertPath,
				PkPath:     frConfig.TlsPkPath},
		}
		err := frontend.Listen()
		if err != nil {
			log.Error().Msgf("error occurred when listening frontend socket:%+v", err)
		}
	}
}
