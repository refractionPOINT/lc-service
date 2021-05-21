package common

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// Abstraction of a Request for Resources.
// Use the `RequestEvent.AsResourceRequest()` to
// generate this structure from a Request.
type ResourceRequest struct {
	isSingleRes   bool
	inIncludeData bool
	ResourceNames []string
}

type ResourceResponse struct {
	// Use `ResourceResponse.FromData()` to populate this
	// structure more conveniently from raw data.

	// LimaCharlie Resource category.
	Category string
	// Sha256 of the Resource data.
	Hash string
	// Base64 encoded Resource data.
	Data string
}

// Load a ResourceResponse struct from a literal buffer.
func NewResourceFromData(category string, data []byte) *ResourceResponse {
	rs := &ResourceResponse{}
	rs.Category = category
	h := sha256.Sum256(data)
	rs.Hash = hex.EncodeToString(h[:])
	rs.Data = base64.StdEncoding.EncodeToString(data)
	return rs
}

type singleResourceRequest struct {
	Name       string `json:"resource" msgpack:"resource"`
	IsWithData bool   `json:"is_include_data" msgpack:"is_include_data"`
}
type multiResourceRequest struct {
	Names      []string `json:"resource" msgpack:"resource"`
	IsWithData bool     `json:"is_include_data" msgpack:"is_include_data"`
}

// Generate a simplified Response from a ResourceRequest.
func (rr ResourceRequest) SupplyResponse(resources map[string]*ResourceResponse) Response {
	if rr.isSingleRes && len(resources) > 1 {
		panic("requested 1 resource, multiple returned")
	}
	if rr.isSingleRes {
		if len(resources) == 0 {
			return Response{
				Data: Dict{"error": "resource not available"},
			}
		}
		found := Dict{}
		for _, v := range resources {
			found["hash"] = v.Hash
			found["res_cat"] = v.Category
			if rr.inIncludeData {
				found["res_data"] = v.Data
			}
		}
		return Response{
			IsSuccess: true,
			Data:      found,
		}
	} else {
		if len(resources) != len(rr.ResourceNames) {
			return Response{
				Data: Dict{"error": "resource not available"},
			}
		}
		res := Dict{}
		for k, v := range resources {
			found := Dict{}
			foundName := k
			found["hash"] = v.Hash
			found["res_cat"] = v.Category
			if rr.inIncludeData {
				found["res_data"] = v.Data
			}
			res[foundName] = found
		}
		return Response{
			IsSuccess: true,
			Data: Dict{
				"resources": res,
			},
		}
	}
}
