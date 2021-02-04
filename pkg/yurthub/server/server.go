/*
Copyright 2020 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"net/http"

	"github.com/alibaba/openyurt/cmd/yurthub/app/config"
	"github.com/alibaba/openyurt/pkg/yurthub/certificate/interfaces"
	"github.com/alibaba/openyurt/pkg/yurthub/profile"
	"github.com/alibaba/openyurt/pkg/yurthub/util"
	"github.com/gorilla/mux"

	"k8s.io/klog"
)

// Server is an interface for providing http service for yurthub
type Server interface {
	Run()
}

type yurtHubServer struct {
	mux            *mux.Router
	certificateMgr interfaces.YurtCertificateManager
	proxyHandler   http.Handler
	cfg            *config.YurtHubConfiguration
}

// NewYurtHubServer creates a Server object
func NewYurtHubServer(cfg *config.YurtHubConfiguration,
	certificateMgr interfaces.YurtCertificateManager,
	proxyHandler http.Handler) Server {
	return &yurtHubServer{
		mux:            mux.NewRouter(),
		certificateMgr: certificateMgr,
		proxyHandler:   proxyHandler,
		cfg:            cfg,
	}
}

func (s *yurtHubServer) Run() {
	var err error
	s.registerHandler()

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.cfg.YurtHubHost, s.cfg.YurtHubPort),
		Handler: s.mux,
	}

	if s.isServeTLS() {
		err = server.ListenAndServeTLS(s.cfg.ServerTLSCert, s.cfg.ServerTLSKey)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil {
		panic(err)
	}
}

func (s *yurtHubServer) isServeTLS() bool {

	existsCert, err := util.FileExists(s.cfg.ServerTLSCert)
	if err != nil {
		klog.Infof("fail to init tls cert file, %v", err)
		return false
	}

	existsKey, err := util.FileExists(s.cfg.ServerTLSKey)
	if err != nil {
		klog.Infof("fail to init tls key file, %v", err)
		return false
	}

	if existsCert && existsKey {
		klog.Infof("use %s cert file and %s key file to run server.", s.cfg.ServerTLSCert, s.cfg.ServerTLSKey)
		return true
	}

	return false
}

func (s *yurtHubServer) registerHandler() {
	// register handlers for update join token
	s.mux.HandleFunc("/v1/token", s.updateToken).Methods("POST", "PUT")

	// register handler for health check
	s.mux.HandleFunc("/v1/healthz", s.healthz).Methods("GET")

	// register handler for profile
	profile.Install(s.mux)

	// attention: "/" route must be put at the end of registerHandler
	// register handlers for proxy to kube-apiserver
	s.mux.PathPrefix("/").Handler(s.proxyHandler)
}

func (s *yurtHubServer) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
