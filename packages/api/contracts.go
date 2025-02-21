/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package api

import (
	"net/http"

	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/model"

	form := &paginatorForm{}
	if err := parseForm(r, form); err != nil {
		errorResponse(w, err, http.StatusBadRequest)
		return
	}

	client := getClient(r)
	logger := getLogger(r)

	contract := &model.Contract{}
	contract.EcosystemID = client.EcosystemID

	count, err := contract.CountByEcosystem()
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("Getting table records count")
		errorResponse(w, err)
		return
	}

	contracts, err := contract.GetListByEcosystem(form.Offset, form.Limit)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting all")
		errorResponse(w, err)
		return
	}

	list := make([]map[string]string, len(contracts))
	for i, c := range contracts {
		list[i] = c.ToMap()
		list[i]["address"] = converter.AddressToString(c.WalletID)
	}

	if len(list) == 0 {
		list = nil
	}

	jsonResponse(w, &listResult{
		Count: count,
		List:  list,
	})
}
