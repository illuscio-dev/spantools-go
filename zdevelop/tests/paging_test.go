package tests

import (
	assert "github.com/stretchr/testify/assert"
	"net/http"
	"spantools/models"
	"testing"
)

func TestPagingReqRoundTrip(test *testing.T) {
	assert := assert.New(test)

	pagingReq := &models.PagingReq{
		Offset: 10,
		Limit:  50,
	}
	
	reqTest := http.Request{
		Header: make(http.Header),
	}

	pagingReq.ToParams(reqTest.Header)
	loaded, err := models.PagingReqFromParams(reqTest.Header, 50)

	assert.Nil(err)
	assert.Equal(pagingReq, loaded)

}

func TestPagingRespRoundTrip(test *testing.T) {
	assert := assert.New(test)

	pagingResp := &models.PagingResp{
		PagingReq:   &models.PagingReq{
			Offset: 10,
			Limit:  50,
		},
		TotalItems:  200,
		TotalPages:  4,
		CurrentPage: 2,
		Next:        "www.api/some/page3",
		Previous:    "www.api/some/page1",
	}

	reqTest := http.Request{
		Header: make(http.Header),
	}

	pagingResp.ToHeaders(reqTest.Header)
	loaded, err := models.PagingRespFromHeaders(reqTest.Header, 50)

	assert.Nil(err)
	assert.Equal(pagingResp, loaded)

}

func TestPagingReqDumpNoLimit(test *testing.T) {
	assert := assert.New(test)

	pagingReq := &models.PagingReq{
		Offset: 10,
		Limit:  0,
	}

	reqTest := http.Request{
		Header: make(http.Header),
	}

	pagingReq.ToParams(reqTest.Header)

	assert.Equal(reqTest.Header.Get("paging-offset"), "10")
	assert.Equal(reqTest.Header.Get("paging-limit"), "")

}

func TestPagingRespOmitNotSets(test *testing.T) {
	assert := assert.New(test)

	pagingResp := models.PagingResp{PagingReq: new(models.PagingReq)}

	reqTest := http.Request{
		Header: make(http.Header),
	}

	pagingResp.ToHeaders(reqTest.Header)

	assert.Equal(reqTest.Header.Get("paging-offset"), "0")
	assert.Equal(reqTest.Header.Get("paging-limit"), "")
	assert.Equal(reqTest.Header.Get("paging-total-items"), "")
	assert.Equal(reqTest.Header.Get("paging-total-pages"), "")
	assert.Equal(reqTest.Header.Get("paging-current-page"), "0")
	assert.Equal(reqTest.Header.Get("paging-next"), "")
	assert.Equal(reqTest.Header.Get("paging-previous"), "")

}

func TestPagingReqLoadLimitDefault(test *testing.T) {
	assert := assert.New(test)

	pagingReq := &models.PagingReq{
		Offset: 10,
		Limit:  0,
	}

	reqTest := http.Request{
		Header: make(http.Header),
	}

	pagingReq.ToParams(reqTest.Header)

	loaded, err := models.PagingReqFromParams(reqTest.Header, 50)

	assert.Nil(err)
	assert.Equal(50, loaded.Limit)

}

func TestPagingNotIntOffset(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-offset", "not an int")
	_, err := models.PagingReqFromParams(reqTest.Header, 50)

	assert.EqualError(err, "paging-offset is not int")

}

func TestPagingNotIntLimit(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-limit", "not an int")
	_, err := models.PagingReqFromParams(reqTest.Header, 50)

	assert.EqualError(err, "paging-limit is not int")

}

func TestPagingTotalItems(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-total-items", "not an int")
	_, err := models.PagingRespFromHeaders(reqTest.Header, 50)

	assert.EqualError(err, "paging-total-items is not int")

}

func TestPagingTotalPages(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-total-pages", "not an int")
	_, err := models.PagingRespFromHeaders(reqTest.Header, 50)

	assert.EqualError(err, "paging-total-pages is not int")

}

func TestPagingCurrentPage(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-current-page", "not an int")
	_, err := models.PagingRespFromHeaders(reqTest.Header, 50)

	assert.EqualError(err, "paging-current-page is not int")

}

func TestPagingLimitRespError(test *testing.T) {
	assert := assert.New(test)

	reqTest := http.Request{
		Header: make(http.Header),
	}

	reqTest.Header.Set("paging-limit", "not an int")
	_, err := models.PagingRespFromHeaders(reqTest.Header, 50)

	assert.EqualError(err, "paging-limit is not int")

}
