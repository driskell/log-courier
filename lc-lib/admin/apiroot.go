/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package admin

import (
	"net/url"

	"github.com/driskell/log-courier/lc-lib/admin/api"
	"github.com/driskell/log-courier/lc-lib/core"
)

type apiRoot struct {
	api.Node
	debug api.Navigatable
}

func (r *apiRoot) Get(path string) (api.Navigatable, error) {
	// Debug is only available via direct request
	if path == "debug" {
		return r.debug, nil
	}

	return r.Node.Get(path)
}

func newAPIRoot(app *core.App) *apiRoot {
	root := &apiRoot{
		debug: api.NewDataEntry(&apiDebug{}),
	}

	root.SetEntry("version", api.NewDataEntry(api.String(core.LogCourierVersion)))
	root.SetEntry("reload", api.NewCallbackEntry(func(values url.Values) (string, error) {
		if err := app.ReloadConfig(); err != nil {
			return "", err
		}
		return "Successfully reloaded configuration", nil
	}))
	return root
}
