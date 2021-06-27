package server

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/edgelesssys/marblerun/coordinator/core"
)

// @title Marblerun Coordinator Client API
// @version 0.3.3
// @description The Client API is designed as an HTTP-REST interface. Responses follow the [JSend](https://github.com/omniti-labs/jsend) style.

// @contact.name Marblerun support
// @contact.url https://www.marblerun.sh/community/
// @contact.email contact@edgeless.systems

// @license.name Apache Mozilla Public License, v. 2.0
// @license.url https://mozilla.org/MPL/2.0/

type ClientAPIServer struct {
	cc core.ClientCore
}

// @tags status
// @router /status [get]
// @summary Get the current status of the Coordinator.
// @description The status indicates the current state of the coordinator, and can be one of the following:
// @description - **0 recovery mode**: Found a sealed state of an old seal key. Waiting for user input on [/recover](https://marblerun.sh/docs/features/recovery/).
// @description - **1 uninitialized**: Fresh start, initializing the Coordinator.
// @description - **2 waiting for a manifest**: Waiting for user input on [/manifest](https://marblerun.sh/docs/workflows/set-manifest/)
// @description - **3 accepting marbles**: Accepting Marbles through the [Marble API](https://marblerun.sh/docs/workflows/add-service/)
// @produce json
// @success 200 {object} GeneralResponse{data=statusResp}
// @failure 405,500 {object} GeneralResponse
func (server *ClientAPIServer) getStatusHandler(w http.ResponseWriter, r *http.Request) {
	statusCode, status, err := server.cc.GetStatus(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, statusResp{statusCode, status})
}

// @tags manifest
// @router /manifest [get]
// @summary Get a SHA-256 of the currently set manifest.
// @description Users can retrieve and inspect the manifest's hash through this endpoint before interacting with the application.
// @description The hash not change when an update manifest has been applied.
// @produce json
// @success 200 {object} GeneralResponse{data=manifestSignatureResp}
// @failure 405,500 {object} GeneralResponse
func (server *ClientAPIServer) getManifestHandler(w http.ResponseWriter, r *http.Request) {
	signature := server.cc.GetManifestSignature(r.Context())
	writeJSON(w, manifestSignatureResp{hex.EncodeToString(signature)})
}

// @tags manifest
// @router /manifest [post]
// @summary Set a manifest.
// @description Before deploying the application to the cluster the manifest needs to be set once by the provider.
// @description On success, an array containing key-value mapping for encrypted secrets to be used for recovering the Coordinator in case of disaster recovery. The key matches each supplied key from RecoveryKeys in the Manifest.
// @accept json
// @Param manifest body manifest.Manifest true "manifest"
// @produce json
// @success 200 {object} GeneralResponse{data=recoveryDataResp}
// @failure 400,405,500 {object} GeneralResponse
func (server *ClientAPIServer) postManifestHandler(w http.ResponseWriter, r *http.Request) {
	manifest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	recoverySecretMap, err := server.cc.SetManifest(r.Context(), manifest)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If recovery data is set, return it
	if len(recoverySecretMap) > 0 {
		secretMap := make(map[string]string, len(recoverySecretMap))
		for name, secret := range recoverySecretMap {
			secretMap[name] = base64.StdEncoding.EncodeToString(secret)
		}
		writeJSON(w, recoveryDataResp{secretMap})
	} else {
		writeJSON(w, nil)
	}
}

// @tags quote
// @router /quote [get]
// @summary Retrieve a remote attestation quote.
// @description For retrieving a remote attestation quote over the whole cluster and the root certificate. The quote is an [SGX-DCAP quote](https://download.01.org/intel-sgx/sgx-dcap/1.9/linux/docs/Intel_SGX_DCAP_ECDSA_Orientation.pdf). Both the provider and the users of the confidential application can use this endpoint to verify the integrity of the Coordinator and the cluster at any time.
// @accept json
// @produce json
// @success 200 {object} GeneralResponse{data=certQuoteResp}
// @failure 405,500 {object} GeneralResponse
func (server *ClientAPIServer) getQuoteHandler(w http.ResponseWriter, r *http.Request) {
	cert, quote, err := server.cc.GetCertQuote(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, certQuoteResp{cert, quote})
}

// @tags recover
// @router /recover [post]
// @summary  Recover the Coordinator when unsealing the existing state fails.
// @description This API endpoint is only available when the coordinator is in recovery mode. Before you can use the endpoint, you need to decrypt the recovery secret which you may have received when setting the manifest initially.
// @description See [Recovering the Coordinator](https://marblerun.sh/docs/workflows/recover-coordinator/) to retrieve the recovery key needed to use this API endpoint correctly.
// @accept json
// @produce json
// @success 200 {object} GeneralResponse{data=recoveryStatusResp}
// @failure 405,500 {object} GeneralResponse
func (server *ClientAPIServer) postRecoverHandler(w http.ResponseWriter, r *http.Request) {
	key, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Perform recover and receive amount of remaining secrets (for multi-party recovery)
	remaining, err := server.cc.Recover(r.Context(), key)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Construct status message based on remaining keys
	var statusMessage string
	if remaining != 0 {
		statusMessage = fmt.Sprintf("Secret was processed successfully. Upload the next secret. Remaining secrets: %d", remaining)
	} else {
		statusMessage = "Recovery successful."
	}

	writeJSON(w, recoveryStatusResp{statusMessage})
}

// @tags update
// @router /update [post]
// @summary  Update the packages specified in the currently set Manifest.
// @description This API endpoint only works when admins were defined in the Manifest.
// @description For more information, look up [Updating a Manifest](https://marblerun.sh/docs/workflows/update-manifest/).
// @accept json
// @produce json
// @success 200 {object} GeneralResponse
// @failure 400,405,500 {object} GeneralResponse
func (server *ClientAPIServer) postUpdateHandler(w http.ResponseWriter, r *http.Request) {
	user := verifyUser(w, r, server.cc)
	if user == nil {
		return
	}

	updateManifest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = server.cc.UpdateManifest(r.Context(), updateManifest, user)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, nil)
}

// @tags secret
// @router /secret [post]
// @summary
// @description
// @accept json
// @produce json
// @success 200 {object} GeneralResponse
// @failure 400,405,500 {object} GeneralResponse
func (server *ClientAPIServer) postSecretHandler(w http.ResponseWriter, r *http.Request) {
	user := verifyUser(w, r, server.cc)
	if user == nil {
		return
	}

	secretManifest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := server.cc.WriteSecrets(r.Context(), secretManifest, user); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, nil)
}

// @tags secret
// @router /secret [get]
// @summary
// @description
// @produce json
// @success 200 {object} GeneralResponse
// @failure 400,405 {object} GeneralResponse
func (server *ClientAPIServer) getSecretHandler(w http.ResponseWriter, r *http.Request) {
	user := verifyUser(w, r, server.cc)
	if user == nil {
		return
	}

	// Secrets are requested via the query string in the form of ?s=<secret_one>&s=<secret_two>&s=...
	requestedSecrets := r.URL.Query()["s"]
	if len(requestedSecrets) <= 0 {
		writeJSONError(w, "invalid query", http.StatusBadRequest)
		return
	}
	for _, req := range requestedSecrets {
		if len(req) <= 0 {
			writeJSONError(w, "malformed query string", http.StatusBadRequest)
			return
		}
	}
	response, err := server.cc.GetSecrets(r.Context(), requestedSecrets, user)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, response)
}

func (server *ClientAPIServer) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, "", http.StatusMethodNotAllowed)
}
