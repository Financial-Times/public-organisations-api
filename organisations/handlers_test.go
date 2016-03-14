package organisations

import(
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
	"github.com/gorilla/mux"
)
var (
	server   *httptest.Server
	organisationsURL string
	isFound bool
)
const (
	expectedCacheControlHeader string = "special header"
)

type mockOrganisationDriver struct {}

func (driver mockOrganisationDriver) Read(id string) (organisation Organisation, found bool, err error) {
	return Organisation{}, isFound, nil
}

func (driver mockOrganisationDriver) CheckConnectivity()  error{
	return nil
}

func init()  {
	OrganisationDriver = mockOrganisationDriver{}
	CacheControlHeader = expectedCacheControlHeader
	r:= mux.NewRouter()
	r.HandleFunc("/organisations/{uuid}", GetOrganisation).Methods("GET")
	server = httptest.NewServer(r)
	organisationsURL = fmt.Sprintf("%s/organisations", server.URL) //Grab the address for the API endpoint
	isFound = true
}

func TestHeadersOKOnFound(t *testing.T) {
	assert := assert.New(t)
	isFound = true
	req, _ := http.NewRequest("GET", organisationsURL + "/00000000-0000-002a-0000-00000000002a", nil)
	res, err := http.DefaultClient.Do(req)
	assert.NoError(err)
	assert.EqualValues(200, res.StatusCode)
	assert.Equal(expectedCacheControlHeader, res.Header.Get("Cache-Control"))
	assert.Equal("application/json; charset=UTF-8", res.Header.Get("Content-Type"))
}

func TestReturnNotFoundIfOrgNotFound(t *testing.T) {
	assert := assert.New(t)
	isFound = false
	req, _ := http.NewRequest("GET", organisationsURL + "/00000000-0000-002a-0000-00000000002a", nil)
	res, err := http.DefaultClient.Do(req)
	assert.NoError(err)
	assert.EqualValues(404, res.StatusCode)
	assert.Equal("application/json; charset=UTF-8", res.Header.Get("Content-Type"))
}