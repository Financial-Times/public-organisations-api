package organisations

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	server           *httptest.Server
	organisationsURL string
	isFound          bool
)

const (
	expectedCacheControlHeader string = "special header"
	canonicalUUID              string = "00000000-0000-002a-0000-00000000002a"
	alternateUUID              string = "00000000-0000-002a-0000-00000000002b"
)

type mockOrganisationDriver struct{}

type mockHTTPClient struct {
	resp       string
	statusCode int
	err        error
}

func (mhc *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(mhc.resp)))
	return &http.Response{Body: cb, StatusCode: mhc.statusCode}, mhc.err
}

func (driver mockOrganisationDriver) Read(id string) (organisation Organisation, found bool, err error) {
	return Organisation{Thing: Thing{ID: canonicalUUID, APIURL: ""}}, isFound, nil
}

func (driver mockOrganisationDriver) CheckConnectivity() error {
	return nil
}

func init() {
	OrganisationDriver = mockOrganisationDriver{}
	CacheControlHeader = expectedCacheControlHeader
	r := mux.NewRouter()
	r.HandleFunc("/organisations/{uuid}", GetOrganisation).Methods("GET")
	server = httptest.NewServer(r)
	organisationsURL = fmt.Sprintf("%s/organisations", server.URL) //Grab the address for the API endpoint
	isFound = true
}

func TestHeadersOKOnFoundForCanonicalNode(t *testing.T) {
	assert := assert.New(t)
	isFound = true
	req, _ := http.NewRequest("GET", organisationsURL+"/"+canonicalUUID, nil)
	res, err := http.DefaultClient.Do(req)
	assert.NoError(err)
	assert.EqualValues(200, res.StatusCode)
	assert.Equal(expectedCacheControlHeader, res.Header.Get("Cache-Control"))
	assert.Equal("application/json; charset=UTF-8", res.Header.Get("Content-Type"))
}

func noRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("Don't redirect!")
}

func TestRedirectHappensOnFoundForAlternateNode(t *testing.T) {
	assert := assert.New(t)
	isFound = true
	req, _ := http.NewRequest("GET", organisationsURL+"/"+alternateUUID, nil)
	client := &http.Client{
		CheckRedirect: noRedirect,
	}
	res, err := client.Do(req)
	assert.Contains(err.Error(), "Don't redirect!")
	assert.EqualValues(301, res.StatusCode)
	assert.Equal("/organisations/"+canonicalUUID, res.Header.Get("Location"))
	assert.Equal("application/json; charset=UTF-8", res.Header.Get("Content-Type"))
}

func TestReturnNotFoundIfOrgNotFound(t *testing.T) {
	assert := assert.New(t)
	isFound = false
	req, _ := http.NewRequest("GET", organisationsURL+"/"+canonicalUUID, nil)
	res, err := http.DefaultClient.Do(req)
	assert.NoError(err)
	assert.EqualValues(404, res.StatusCode)
	assert.Equal("application/json; charset=UTF-8", res.Header.Get("Content-Type"))
}
