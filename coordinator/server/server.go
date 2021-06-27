// Copyright (c) Edgeless Systems GmbH.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package server contains the ClientAPI HTTP-REST and MarbleAPI gRPC server.
package server

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"os"

	"github.com/edgelesssys/marblerun/coordinator/core"
	"github.com/edgelesssys/marblerun/coordinator/rpc"
	"github.com/edgelesssys/marblerun/coordinator/user"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GeneralResponse is a wrapper for all our REST API responses to follow the JSend style: https://github.com/omniti-labs/jsend
type GeneralResponse struct {
	Status  string      `json:"status" example:"success"`                  // success or error
	Data    interface{} `json:"data"`                                      // may be null, allways null for errors
	Message string      `json:"message,omitempty" example:"error message"` // only used for errors
}
type certQuoteResp struct {
	Cert  string
	Quote []byte
}
type statusResp struct {
	StatusCode    int    `example:"2"`
	StatusMessage string `example:"Coordinator is ready to accept a manifest."`
}
type manifestSignatureResp struct {
	ManifestSignature string `example:"3fff78e99dd9bd801e0a3a22b7f7a24a492302c4d00546d18c7f7ed6e26e95c3"`
}

// Contains RSA-encrypted AES state sealing key with public key specified by user in manifest
type recoveryDataResp struct {
	RecoverySecrets map[string]string
}

type recoveryStatusResp struct {
	StatusMessage string
}

// RunMarbleServer starts a gRPC with the given Coordinator core.
// `address` is the desired TCP address like "localhost:0".
// The effective TCP address is returned via `addrChan`.
func RunMarbleServer(core *core.Core, addr string, addrChan chan string, errChan chan error, zapLogger *zap.Logger) {
	tlsConfig := tls.Config{
		GetCertificate: core.GetTLSMarbleRootCertificate,
		// NOTE: we'll verify the cert later using the given quote
		ClientAuth: tls.RequireAnyClientCert,
	}
	creds := credentials.NewTLS(&tlsConfig)

	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpc_zap.ReplaceGrpcLoggerV2(zapLogger)

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zapLogger),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLogger),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	)

	rpc.RegisterMarbleServer(grpcServer, core)
	socket, err := net.Listen("tcp", addr)
	if err != nil {
		errChan <- err
		return
	}
	addrChan <- socket.Addr().String()
	err = grpcServer.Serve(socket)
	if err != nil {
		errChan <- err
	}
}

// CreateServeMux creates a mux that serves the client API.
func CreateServeMux(cc core.ClientCore) *mux.Router {
	router := mux.NewRouter()
	server := ClientAPIServer{cc}
	router.HandleFunc("/status", server.getStatusHandler).Methods("GET")
	router.HandleFunc("/status", server.methodNotAllowedHandler)
	router.HandleFunc("/manifest", server.getManifestHandler).Methods("GET")
	router.HandleFunc("/manifest", server.postManifestHandler).Methods("POST")
	router.HandleFunc("/manifest", server.methodNotAllowedHandler)
	router.HandleFunc("/quote", server.getQuoteHandler).Methods("GET")
	router.HandleFunc("/quote", server.methodNotAllowedHandler)
	router.HandleFunc("/recover", server.postRecoverHandler).Methods("POST")
	router.HandleFunc("/recover", server.methodNotAllowedHandler)
	router.HandleFunc("/update", server.postUpdateHandler).Methods("POST")
	router.HandleFunc("/update", server.methodNotAllowedHandler)
	router.HandleFunc("/secrets", server.postSecretHandler).Methods("POST")
	router.HandleFunc("/secrets", server.getSecretHandler).Methods("GET")
	router.HandleFunc("/secrets", server.methodNotAllowedHandler)
	return router
}

func verifyUser(w http.ResponseWriter, r *http.Request, cc core.ClientCore) *user.User {
	// Abort if no user client certificate was provided
	if r.TLS == nil {
		writeJSONError(w, "no client certificate provided", http.StatusUnauthorized)
		return nil
	}
	verifiedUser, err := cc.VerifyUser(r.Context(), r.TLS.PeerCertificates)
	if err != nil {
		writeJSONError(w, "unauthorized user", http.StatusUnauthorized)
		return nil
	}
	return verifiedUser
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	dataToReturn := GeneralResponse{Status: "success", Data: v}
	if err := json.NewEncoder(w).Encode(dataToReturn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSONError(w http.ResponseWriter, errorString string, httpErrorCode int) {
	marshalledJSON, err := json.Marshal(GeneralResponse{Status: "error", Message: errorString})
	// Only fall back to non-JSON error when we cannot even marshal the error (which is pretty bad)
	if err != nil {
		http.Error(w, errorString, httpErrorCode)
	}
	http.Error(w, string(marshalledJSON), httpErrorCode)
}

// RunClientServer runs a HTTP server serving mux.
func RunClientServer(mux *mux.Router, address string, tlsConfig *tls.Config, zapLogger *zap.Logger) {
	loggedRouter := handlers.LoggingHandler(os.Stdout, mux)
	server := http.Server{
		Addr:      address,
		Handler:   loggedRouter,
		TLSConfig: tlsConfig,
	}
	zapLogger.Info("starting client https server", zap.String("address", address))
	err := server.ListenAndServeTLS("", "")
	zapLogger.Warn(err.Error())
}

// RunPrometheusServer runs a HTTP server handling the prometheus metrics endpoint
func RunPrometheusServer(address string, zapLogger *zap.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	zapLogger.Info("starting prometheus /metrics endpoint", zap.String("address", address))
	err := http.ListenAndServe(address, mux)
	zapLogger.Warn(err.Error())
}
