package common

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestResourceConvertion(t *testing.T) {
	// Single resource request
	r, err := RequestEvent{
		Data: Dict{
			"resource":        "test1",
			"is_include_data": true,
		},
	}.AsResourceRequest()
	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}
	if !r.inIncludeData {
		t.Errorf("wrong isIncludeData: %+v", r)
	}
	if !r.isSingleRes {
		t.Errorf("wrong isSingleRes: %+v", r)
	}
	if len(r.ResourceNames) != 1 || r.ResourceNames[0] != "test1" {
		t.Errorf("wrong ResourceNames: %+v", r)
	}

	// Multi resource request
	r, err = RequestEvent{
		Data: Dict{
			"resource":        []string{"test1", "test2"},
			"is_include_data": true,
		},
	}.AsResourceRequest()
	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}
	if !r.inIncludeData {
		t.Errorf("wrong isIncludeData: %+v", r)
	}
	if r.isSingleRes {
		t.Errorf("wrong isSingleRes: %+v", r)
	}
	if len(r.ResourceNames) != 2 || r.ResourceNames[0] != "test1" || r.ResourceNames[1] != "test2" {
		t.Errorf("wrong ResourceNames: %+v", r)
	}
}

func TestResourceResponse(t *testing.T) {
	resData := []byte("thisisatest")
	h := sha256.Sum256(resData)
	resHash := hex.EncodeToString(h[:])
	resEncoded := base64.StdEncoding.EncodeToString(resData)

	r := NewResourceFromData("lookup", resData)
	if r == nil {
		t.Error("bad load")
	}
	if r.Category != "lookup" {
		t.Errorf("wrong cat: %+v", r)
	}
	if r.Data != resEncoded {
		t.Errorf("wrong dat: %+v", r)
	}
	if r.Hash != resHash {
		t.Errorf("wrong hash: %+v", r)
	}
}

func TestResourceSupply(t *testing.T) {
	// Single resource
	r, err := RequestEvent{
		Data: Dict{
			"resource":        []string{"test1", "test2"},
			"is_include_data": true,
		},
	}.AsResourceRequest()

	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}

	s := map[string]*ResourceResponse{
		"test1": NewResourceFromData("lookup", []byte("data1")),
		"test2": NewResourceFromData("lookup", []byte("data2")),
	}

	resp := r.SupplyResponse(s)
	if fmt.Sprintf("%+v", resp) != `{IsSuccess:true IsRetriable:false Error: Data:map[resources:map[test1:map[hash:5b41362bc82b7f3d56edc5a306db22105707d01ff4819e26faef9724a2d406c9 res_cat:lookup res_data:ZGF0YTE=] test2:map[hash:d98cf53e0c8b77c14a96358d5b69584225b4bb9026423cbc2f7b0161894c402c res_cat:lookup res_data:ZGF0YTI=]]] Jobs:[]}` {
		t.Errorf("unexpected supply: %+v", resp)
	}

	// Multi resource
	r, err = RequestEvent{
		Data: Dict{
			"resource":        "test1",
			"is_include_data": true,
		},
	}.AsResourceRequest()

	if err != nil {
		t.Errorf("AsResourceRequest: %v", err)
	}

	s = map[string]*ResourceResponse{
		"test1": NewResourceFromData("lookup", []byte("data1")),
	}

	resp = r.SupplyResponse(s)
	if fmt.Sprintf("%+v", resp) != `{IsSuccess:true IsRetriable:false Error: Data:map[hash:5b41362bc82b7f3d56edc5a306db22105707d01ff4819e26faef9724a2d406c9 res_cat:lookup res_data:ZGF0YTE=] Jobs:[]}` {
		t.Errorf("unexpected supply: %+v", resp)
	}
}
