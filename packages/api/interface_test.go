/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package api

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInterfaceRow(t *testing.T) {
	cases := []struct {
		url        string
		contract   string
		equalAttrs []string
	}{
		{"interface/page/", "NewPage", []string{"Name", "Value", "Menu", "Conditions"}},
		{"interface/menu/", "NewMenu", []string{"Name", "Value", "Title", "Conditions"}},
		{"interface/block/", "NewBlock", []string{"Name", "Value", "Conditions"}},
	}

	checkEqualAttrs := func(form url.Values, result map[string]interface{}, equalKeys []string) {
		for _, key := range equalKeys {
			v := result[strings.ToLower(key)]
			assert.EqualValues(t, form.Get(key), v)
		}
	}

	errUnauthorized := `401 {"error": "E_UNAUTHORIZED", "msg": "Unauthorized" }`
	for _, c := range cases {
		assert.EqualError(t, sendGet(c.url+"-", &url.Values{}, nil), errUnauthorized)
		assert.NoError(t, sendGet(c.url+name, &url.Values{}, &result))
		checkEqualAttrs(form, result, c.equalAttrs)
	}
}

func TestNewMenuNoError(t *testing.T) {
	require.NoError(t, keyLogin(1))
	menuname := "myTestMenu"
	form := url.Values{"Name": {menuname}, "Value": {`first
		second
		third`}, "Title": {`My Test Menu`},
		"Conditions": {`true`}}
	assert.NoError(t, postTx(`NewMenu`, &form))

	err := postTx(`NewMenu`, &form)
	assert.Equal(t, fmt.Sprintf(`{"type":"warning","error":"Menu %s already exists"}`, menuname), cutErr(err))
}

func TestEditMenuNoError(t *testing.T) {
	require.NoError(t, keyLogin(1))
	form := url.Values{
		"Id": {"1"},
		"Value": {`first
		second
		third
		andmore`},
		"Title": {`My edited Test Menu`},
	}
	assert.NoError(t, postTx(`EditMenu`, &form))
}

func TestAppendMenuNoError(t *testing.T) {
	require.NoError(t, keyLogin(1))
	form := url.Values{
		"Id":    {"3"},
		"Value": {"appended item"},
	}

	assert.NoError(t, postTx("AppendMenu", &form))
}
