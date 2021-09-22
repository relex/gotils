// Copyright 2021 RELEX Oy
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package promreg

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/relex/gotils/logger"
)

// LaunchMetricListener starts a HTTP server for Prometheus metrics and optionally /debug/pprof
//
// If the address contains unspecified port (":0"), a random port is assigned and set to server.Addr
func LaunchMetricListener(address string, gatherer prometheus.Gatherer, enablePprof bool) *http.Server {
	mlogger := logger.WithField("component", "MetricListener")

	lsnr, lsnrErr := net.Listen("tcp", address)
	if lsnrErr != nil {
		mlogger.Fatal("failed to listen for metrics: ", lsnrErr)
	}
	mlogger.Infof("listening on %s for metrics...", lsnr.Addr())

	mux := createServerMux(gatherer)
	if enablePprof {
		registerPprocHandlers(mux)
	}

	srv := &http.Server{}
	srv.Addr = lsnr.Addr().String()
	srv.Handler = mux

	go func() {
		if err := srv.Serve(lsnr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mlogger.Error("failed to serve metric listener: ", err)
		}
	}()

	return srv
}

func createServerMux(gatherer prometheus.Gatherer) *http.ServeMux {
	mux := http.NewServeMux()

	mhandler := promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
	)
	mux.Handle("/metrics", mhandler)
	mux.Handle("/api/v1/metrics/prometheus", mhandler) // for fluent-bit compatibility

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		appName := filepath.Base(os.Args[0])
		fmt.Fprintf(w, `
<html>
	<head>
		<title>%s metric listener</title>
	</head>
	<body>
		<h1>Metric listener for %s</h1>
		<ul>
			<li><a href='/debug/pprof'>/debug/pprof</a></li>
			<li><a href='/metrics'>/metrics</a></li>
		</ul>
	</body>
</html>`, appName, appName)
	})

	return mux
}

func registerPprocHandlers(mux *http.ServeMux) {
	// see pprof.init()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
