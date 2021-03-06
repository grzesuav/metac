/*
Copyright 2019 Google Inc.
Copyright 2019 The MayaData Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

// ServeWebhook is a helper for quickly creating a
// webhook server in tests.
func (f *Fixture) ServeWebhook(
	reconciler func(request []byte) (response []byte, err error),
) *httptest.Server {
	// create a new instance of http test server
	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(
					w,
					fmt.Sprintf("Unsupported method: %s", r.Method),
					http.StatusMethodNotAllowed,
				)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(
					w,
					fmt.Sprintf("Can't read body: %v", err),
					http.StatusBadRequest,
				)
				return
			}
			// invoke reconciler here
			resp, err := reconciler(body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}),
	)
	f.addToTeardown(func() error {
		srv.Close()
		return nil
	})
	return srv
}
