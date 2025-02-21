/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package api

import (

	form := &ecosystemForm{
		Validator: m.EcosysIDValidator,
	}

	if err := parseForm(r, form); err != nil {
		errorResponse(w, err, http.StatusBadRequest)
		return
	}

	params := mux.Vars(r)

	sp := &model.StateParameter{}
	sp.SetTablePrefix(form.EcosystemPrefix)
	name := params["name"]

	if found, err := sp.Get(nil, name); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("Getting state parameter by name")
		errorResponse(w, err)
		return
	} else if !found {
		logger.WithFields(log.Fields{"type": consts.NotFound, "key": name}).Error("state parameter not found")
		errorResponse(w, errParamNotFound.Errorf(name))
		return
	}

	jsonResponse(w, &paramResult{
		ID:         converter.Int64ToStr(sp.ID),
		Name:       sp.Name,
		Value:      sp.Value,
		Conditions: sp.Conditions,
	})
}

func getEcosystemNameHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	ecosystemID := converter.StrToInt64(r.FormValue("id"))
	ecosystems := model.Ecosystem{}
	found, err := ecosystems.Get(nil, ecosystemID)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("on getting ecosystem name")
		errorResponse(w, err)
		return
	}
	if !found {
		logger.WithFields(log.Fields{"type": consts.NotFound, "ecosystem_id": ecosystemID}).Error("ecosystem by id not found")
		errorResponse(w, errParamNotFound.Errorf("name"))
		return
	}

	jsonResponse(w, &struct {
		EcosystemName string `json:"ecosystem_name"`
	}{
		EcosystemName: ecosystems.Name,
	})
}
