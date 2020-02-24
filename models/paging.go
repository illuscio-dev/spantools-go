package models

import (
	"golang.org/x/xerrors"
	"strconv"
)

type valueSetter interface {
	Set(key string, value string)
}

// Paging parameters for request.
type PagingReq struct {
	// How far to offset the page.
	Offset int
	// Maximum item count to return.
	Limit int
}

// Dumps paging information to request URL params.
func (pagingReq *PagingReq) ToParams(params valueSetter) {
	params.Set("paging-offset", strconv.Itoa(pagingReq.Offset))
	// Only send back limit if it is valid.
	if pagingReq.Limit > 0 {
		params.Set("paging-limit", strconv.Itoa(pagingReq.Limit))
	}
}

type PagingResp struct {
	*PagingReq
	TotalItems  int
	TotalPages  int
	CurrentPage int
	Next        string
	Previous    string
}

func (pagingResp *PagingResp) ToHeaders(headers valueSetter) {
	pagingResp.PagingReq.ToParams(headers)
	// Only send back valid fields.
	if pagingResp.TotalItems > 0 {
		headers.Set("paging-total-items", strconv.Itoa(pagingResp.TotalItems))
	}
	if pagingResp.TotalPages > 0 {
		headers.Set("paging-total-pages", strconv.Itoa(pagingResp.TotalPages))
	}
	if pagingResp.CurrentPage > -1 {
		headers.Set("paging-current-page", strconv.Itoa(pagingResp.CurrentPage))
	}
	if pagingResp.Previous != "" {
		headers.Set("paging-previous", pagingResp.Previous)
	}
	if pagingResp.Next != "" {
		headers.Set("paging-next", pagingResp.Next)
	}
}

type valueFetcher interface {
	Get(key string) string
}

func getInt(headers valueFetcher, fieldName string, defaultValue int) (int, error) {
	var valueInt int
	var err error

	value := headers.Get(fieldName)
	if value == "" {
		valueInt = defaultValue
	} else {
		valueInt, err = strconv.Atoi(value)
		if err != nil {
			return 0, xerrors.New(fieldName + " is not int")
		}
	}
	return valueInt, nil
}

// Generates a PagingReq object from request parameters.
func PagingReqFromParams(
	headers valueFetcher, defaultLimit int,
) (pagingReq *PagingReq, err error) {
	pagingReq = &PagingReq{}

	pagingReq.Offset, err = getInt(headers, "paging-offset", 0)
	if err != nil {
		return nil, err
	}

	pagingReq.Limit, err = getInt(headers, "paging-limit", defaultLimit)
	if err != nil {
		return nil, err
	}

	return pagingReq, nil
}

// PagingRespFromHeaders generates a PagingResp object from response headers.
func PagingRespFromHeaders(
	params valueFetcher, defaultLimit int,
) (pagingResp *PagingResp, err error) {

	pagingReq, err := PagingReqFromParams(params, defaultLimit)
	if err != nil {
		return nil, err
	}

	pagingResp = &PagingResp{PagingReq: pagingReq}

	// These fields may not always have valid values. For this reason, we are going
	// to use -1 as a default to flag that the value was not present in the params.
	pagingResp.TotalPages, err = getInt(
		params, "paging-total-pages", -1,
	)
	if err != nil {
		return nil, err
	}

	pagingResp.TotalItems, err = getInt(
		params, "paging-total-items", -1,
	)
	if err != nil {
		return nil, err
	}

	pagingResp.CurrentPage, err = getInt(
		params, "paging-current-page", -1,
	)
	if err != nil {
		return nil, err
	}

	pagingResp.Previous = params.Get("paging-previous")
	pagingResp.Next = params.Get("paging-next")

	return pagingResp, nil
}
