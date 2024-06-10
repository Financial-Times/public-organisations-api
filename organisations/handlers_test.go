package organisations

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	expectedCacheControlHeader string = "special header"
)

type mockHTTPClient struct {
	resp       string
	statusCode int
	err        error
}

type testCase struct {
	name         string
	url          string
	clientCode   int
	clientBody   string
	clientError  error
	expectedCode int
	expectedBody string
}

func init() {

}

func (mhc *mockHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	cb := ioutil.NopCloser(bytes.NewReader([]byte(mhc.resp)))
	return &http.Response{Body: cb, StatusCode: mhc.statusCode}, mhc.err
}

func TestHandlers(t *testing.T) {
	log := logger.NewUPPInfoLogger("tests")
	var mockClient mockHTTPClient
	router := mux.NewRouter()

	invalidUUID := testCase{
		"Get organisation - Invalid UUID results in error",
		"/organisations/1234",
		200,
		getBasicOrganisationAsConcept,
		nil,
		400,
		`{"message": "uuid '1234' is either missing or invalid"}`,
	}
	conceptApiError := testCase{
		"Get organisations - Concepts API Error results in error",
		"/organisations/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		503,
		"",
		errors.New("Downstream error"),
		500,
		`{"message": "failed to return organisation"}`,
	}
	redirectedUUID := testCase{
		"Get organisations - Given UUID was not canonical",
		"/organisations/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		200,
		getRedirectedOrganisation,
		nil,
		301,
		``,
	}
	errorOnInvalidJson := testCase{
		"Get organisations - Error on invalid json",
		"/organisations/52aa645b-79d6-4f6f-910b-e1cff3f25a15",
		200,
		`{`,
		nil,
		500,
		`{"message": "failed to return organisation"}`,
	}
	notFound := testCase{
		"Get organisation - not found",
		"/organisations/2d3e16e0-61cb-4322-8aff-3b01c59f4daa",
		404,
		"",
		nil,
		404,
		`{"message": "organisation not found"}`,
	}
	nonOrganisationsReturnsNotFound := testCase{
		"Get organisation - Other type returns not found",
		"/organisations/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
		200,
		getPersonAsConcept,
		nil,
		404,
		`{"message": "organisation not found"}`,
	}

	deprecatedConcept := testCase{
		"Get organisation - Retrieves and transforms correctly a deprecated organisation",
		"/organisations/6fc8fbac-b4ee-11e8-a790-6c96cfdf3997",
		200,
		getCompleteDeprecatedOrganisationAsConcept,
		nil,
		200,
		getTransformedCompleteDeprecatedOrganisation,
	}

	successfulRequest := testCase{
		"Get organisation - Retrieves and transforms correctly",
		"/organisations/7c5218a0-3755-463e-abbc-1a1632cfd1da",
		200,
		getCompleteOrganisationAsConcept,
		nil,
		200,
		getTransformedCompleteOrganisation,
	}

	testCases := []testCase{
		invalidUUID,
		conceptApiError,
		redirectedUUID,
		errorOnInvalidJson,
		notFound,
		nonOrganisationsReturnsNotFound,
		successfulRequest,
		deprecatedConcept,
	}

	for _, test := range testCases {
		mockClient.resp = test.clientBody
		mockClient.statusCode = test.clientCode
		mockClient.err = test.clientError
		bh := NewHandler(&mockClient, "localhost:8080/concepts", log)
		bh.RegisterHandlers(router)

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", test.url, nil)

		router.ServeHTTP(rr, req)
		assert.Equal(t, test.expectedCode, rr.Code, test.name+" failed: status codes do not match!")

		if rr.Code == 200 {
			assert.Equal(t, transformBody(test.expectedBody), rr.Body.String(), test.name+" failed: status body does not match!")
			continue
		}
		assert.Equal(t, test.expectedBody, rr.Body.String(), test.name+" failed: status body does not match!")
	}
}

func TestHeadersOKOnFoundForCanonicalNode(t *testing.T) {
	log := logger.NewUPPInfoLogger("tests")
	var mockClient mockHTTPClient
	mockClient.resp = getBasicOrganisationAsConcept
	mockClient.statusCode = 200
	mockClient.err = nil

	CacheControlHeader = expectedCacheControlHeader

	router := mux.NewRouter()
	bh := NewHandler(&mockClient, "localhost:8080/concepts", log)
	bh.RegisterHandlers(router)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/organisations/d6b12f0c-bf3f-4045-a07b-1e4e49103fd6", nil)

	router.ServeHTTP(rec, req)
	fmt.Print(rec.Body)

	assert.Equal(t, expectedCacheControlHeader, rec.Header().Get("Cache-Control"))
	assert.Equal(t, "application/json; charset=UTF-8", rec.Header().Get("Content-Type"))
}

func transformBody(testBody string) string {
	stripNewLines := strings.Replace(testBody, "\n", "", -1)
	stripTabs := strings.Replace(stripNewLines, "\t", "", -1)
	return stripTabs + "\n"
}

var getBasicOrganisationAsConcept = `{
	"id": "http://www.ft.com/thing/d6b12f0c-bf3f-4045-a07b-1e4e49103fd6",
	"apiUrl": "http://api.ft.com/concepts/d6b12f0c-bf3f-4045-a07b-1e4e49103fd6",
	"type": "http://www.ft.com/ontology/organisation/Organisation",
	"prefLabel": "Google Inc"
}`

var getRedirectedOrganisation = `{
	"id": "http://www.ft.com/thing/d6b12f0c-bf3f-4045-a07b-1e4e49103fd6",
	"apiUrl": "http://api.ft.com/concepts/d6b12f0c-bf3f-4045-a07b-1e4e49103fd6",
	"type": "http://www.ft.com/ontology/organisation/Organisation",
	"prefLabel": "Google Inc"
}`

var getPersonAsConcept = `{
	"id": "http://www.ft.com/thing/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
	"apiUrl": "http://api.ft.com/concepts/f92a4ca4-84f9-11e8-8f42-da24cd01f044",
	"type": "http://www.ft.com/ontology/person/Person",
	"prefLabel": "Not a organisation"
}`

var getCompleteOrganisationAsConcept = `{
	"id": "http://www.ft.com/thing/7c5218a0-3755-463e-abbc-1a1632cfd1da",
	"apiUrl": "http://api.ft.com/concepts/7c5218a0-3755-463e-abbc-1a1632cfd1da",
	"type": "http://www.ft.com/ontology/organisation/Organisation",
	"prefLabel": "Nintendo Co Ltd",
	"alternativeLabels": [
		{
			"type": "http://www.ft.com/ontology/formerName",
			"value": "Nintendo Playing Card Co., Ltd."
		},
		{
			"type": "http://www.ft.com/ontology/properName",
			"value": "Nintendo Co., Ltd."
		},
		{
			"type": "http://www.ft.com/ontology/shortName",
			"value": "Nintendo"
		}
	],
	"countryCode": "JP",
	"countryOfIncorporation": "JP",
	"leiCode": "353800FEEXU6I9M0ZF27",
	"postalCode": "601-8116",
	"yearFounded": 1889,
	"relatedConcepts": [
		{
			"concept": {
				"id": "http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
				"apiUrl": "http://api.ft.com/concepts/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
				"type": "http://www.ft.com/ontology/FinancialInstrument",
				"prefLabel": "Nintendo Co., Ltd.",
				"alternativeLabels": [
					{
						"type": "http://www.ft.com/ontology/Alias",
						"value": "Nintendo Co., Ltd."
					}
				],
				"figiCode": "BBG000BLCPP4"
			},
			"predicate": "http://www.ft.com/ontology/issued"
		},
		{
			"concept": {
				"id": "http://api.ft.com/things/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
				"apiUrl": "http://api.ft.com/organisations/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
				"type": "http://www.ft.com/ontology/organisation/Organisation",
				"prefLabel": "Alphabet Inc",
				"countryCode": "US",
				"countryOfIncorporation": "US",
				"postalCode": "94043",
				"yearFounded": 2015
			},
			"predicate": "http://www.ft.com/ontology/subOrganisationOf"
		},
		{
			"concept": {
				"id": "http://api.ft.com/things/1b070fbb-6331-3225-bb57-9108deb67df4",
				"apiUrl": "http://api.ft.com/concepts/1b070fbb-6331-3225-bb57-9108deb67df4",
				"type": "http://www.ft.com/ontology/organisation/Organisation",
				"prefLabel": "Nintendo France SARL",
				"alternativeLabels": [
					{
						"type": "http://www.ft.com/ontology/Alias",
						"value": "Nintendo France SARL"
					}
				],
				"countryOfIncorporation": "FR",
				"postalCode": "95031"
			},
			"predicate": "http://www.ft.com/ontology/parentOrganisationOf"
		}
	]
}`

var getTransformedCompleteOrganisation = `{
	"id":"http://api.ft.com/things/7c5218a0-3755-463e-abbc-1a1632cfd1da",
	"apiUrl":"http://api.ft.com/organisations/7c5218a0-3755-463e-abbc-1a1632cfd1da",
	"prefLabel":"Nintendo Co Ltd",
	"properName":"Nintendo Co., Ltd.",
	"shortName":"Nintendo",
	"formerNames":[
		"Nintendo Playing Card Co., Ltd."
	],
	"countryCode":"JP",
	"countryOfIncorporation":"JP",
	"postalCode":"601-8116",
	"yearFounded":1889,
	"types":[
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/concept/Concept",
		"http://www.ft.com/ontology/organisation/Organisation"
	],
	"directType":"http://www.ft.com/ontology/organisation/Organisation",
	"labels":[
		"Nintendo Playing Card Co., Ltd.",
		"Nintendo Co., Ltd.",
		"Nintendo"
	],
	"leiCode":"353800FEEXU6I9M0ZF27",
	"parentOrganisation":{
		"id":"http://api.ft.com/things/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
		"apiUrl":"http://api.ft.com/organisations/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
		"prefLabel":"Alphabet Inc",
		"types":[
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/organisation/Organisation"
		],
		"directType":"http://www.ft.com/ontology/organisation/Organisation"
	},
	"subsidiaries":[
		{
			"id":"http://api.ft.com/things/1b070fbb-6331-3225-bb57-9108deb67df4",
			"apiUrl":"http://api.ft.com/organisations/1b070fbb-6331-3225-bb57-9108deb67df4",
			"prefLabel":"Nintendo France SARL",
			"types":[
				"http://www.ft.com/ontology/core/Thing",
				"http://www.ft.com/ontology/concept/Concept",
				"http://www.ft.com/ontology/organisation/Organisation"
			],
			"directType":"http://www.ft.com/ontology/organisation/Organisation"
		}
	],
	"financialInstrument":{
		"id":"http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
		"apiUrl":"http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
		"prefLabel":"Nintendo Co., Ltd.",
		"types":[
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/FinancialInstrument"
		],
		"directType":"http://www.ft.com/ontology/FinancialInstrument",
		"FIGI":"BBG000BLCPP4"
	}
}`

var getCompleteDeprecatedOrganisationAsConcept = `{
	"id": "http://www.ft.com/thing/6fc8fbac-b4ee-11e8-a790-6c96cfdf3997",
	"apiUrl": "http://api.ft.com/concepts/6fc8fbac-b4ee-11e8-a790-6c96cfdf3997",
	"type": "http://www.ft.com/ontology/organisation/Organisation",
	"prefLabel": "Nintendo Co Ltd",
	"alternativeLabels": [
		{
			"type": "http://www.ft.com/ontology/formerName",
			"value": "Nintendo Playing Card Co., Ltd."
		},
		{
			"type": "http://www.ft.com/ontology/properName",
			"value": "Nintendo Co., Ltd."
		},
		{
			"type": "http://www.ft.com/ontology/shortName",
			"value": "Nintendo"
		}
	],
	"countryCode": "JP",
	"countryOfIncorporation": "JP",
	"leiCode": "353800FEEXU6I9M0ZF27",
	"postalCode": "601-8116",
	"yearFounded": 1889,
	"isDeprecated": true,
	"relatedConcepts": [
		{
			"concept": {
				"id": "http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
				"apiUrl": "http://api.ft.com/concepts/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
				"type": "http://www.ft.com/ontology/FinancialInstrument",
				"prefLabel": "Nintendo Co., Ltd.",
				"alternativeLabels": [
					{
						"type": "http://www.ft.com/ontology/Alias",
						"value": "Nintendo Co., Ltd."
					}
				],
				"figiCode": "BBG000BLCPP4"
			},
			"predicate": "http://www.ft.com/ontology/issued"
		},
		{
			"concept": {
				"id": "http://api.ft.com/things/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
				"apiUrl": "http://api.ft.com/organisations/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
				"type": "http://www.ft.com/ontology/organisation/Organisation",
				"prefLabel": "Alphabet Inc",
				"countryCode": "US",
				"countryOfIncorporation": "US",
				"postalCode": "94043",
				"yearFounded": 2015
			},
			"predicate": "http://www.ft.com/ontology/subOrganisationOf"
		},
		{
			"concept": {
				"id": "http://api.ft.com/things/1b070fbb-6331-3225-bb57-9108deb67df4",
				"apiUrl": "http://api.ft.com/concepts/1b070fbb-6331-3225-bb57-9108deb67df4",
				"type": "http://www.ft.com/ontology/organisation/Organisation",
				"prefLabel": "Nintendo France SARL",
				"alternativeLabels": [
					{
						"type": "http://www.ft.com/ontology/Alias",
						"value": "Nintendo France SARL"
					}
				],
				"countryOfIncorporation": "FR",
				"postalCode": "95031"
			},
			"predicate": "http://www.ft.com/ontology/parentOrganisationOf"
		}
	]
}`

var getTransformedCompleteDeprecatedOrganisation = `{
	"id":"http://api.ft.com/things/6fc8fbac-b4ee-11e8-a790-6c96cfdf3997",
	"apiUrl":"http://api.ft.com/organisations/6fc8fbac-b4ee-11e8-a790-6c96cfdf3997",
	"prefLabel":"Nintendo Co Ltd",
	"properName":"Nintendo Co., Ltd.",
	"shortName":"Nintendo",
	"formerNames":["Nintendo Playing Card Co., Ltd."],
	"countryCode":"JP",
	"countryOfIncorporation":"JP",
	"postalCode":"601-8116",
	"yearFounded":1889,
	"types":[
		"http://www.ft.com/ontology/core/Thing",
		"http://www.ft.com/ontology/concept/Concept",
		"http://www.ft.com/ontology/organisation/Organisation"
	],
	"directType":"http://www.ft.com/ontology/organisation/Organisation",
	"labels":[
		"Nintendo Playing Card Co., Ltd.",
		"Nintendo Co., Ltd.",
		"Nintendo"
	],
	"leiCode":"353800FEEXU6I9M0ZF27",
	"parentOrganisation":{
		"id":"http://api.ft.com/things/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
		"apiUrl":"http://api.ft.com/organisations/335e9e5a-8f2e-11e8-8f42-da24cd01f044",
		"prefLabel":"Alphabet Inc",
		"types":[
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/organisation/Organisation"
		],
		"directType":"http://www.ft.com/ontology/organisation/Organisation"
	},
	"subsidiaries":[
		{
			"id":"http://api.ft.com/things/1b070fbb-6331-3225-bb57-9108deb67df4",
			"apiUrl":"http://api.ft.com/organisations/1b070fbb-6331-3225-bb57-9108deb67df4",
			"prefLabel":"Nintendo France SARL",
			"types":[
				"http://www.ft.com/ontology/core/Thing",
				"http://www.ft.com/ontology/concept/Concept",
				"http://www.ft.com/ontology/organisation/Organisation"
			],
			"directType":"http://www.ft.com/ontology/organisation/Organisation"
		}
	],
	"financialInstrument":{
		"id":"http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
		"apiUrl":"http://api.ft.com/things/dfee4b8f-ceee-37ba-ab24-752cf7a9281c",
		"prefLabel":"Nintendo Co., Ltd.",
		"types":[
			"http://www.ft.com/ontology/core/Thing",
			"http://www.ft.com/ontology/concept/Concept",
			"http://www.ft.com/ontology/FinancialInstrument"
		],
		"directType":"http://www.ft.com/ontology/FinancialInstrument",
		"FIGI":"BBG000BLCPP4"
	},
	"isDeprecated":true
}`
