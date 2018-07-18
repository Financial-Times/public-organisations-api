package organisations

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/service-status-go/gtg"
	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type HTTPClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type OrganisationsHandler struct {
	client      HTTPClient
	conceptsURL string
}

// OrganisationDriver for cypher queries
var OrganisationDriver Driver
var CacheControlHeader string

func NewHandler(client HTTPClient, conceptsURL string) OrganisationsHandler {
	return OrganisationsHandler{
		client,
		conceptsURL,
	}
}

func (h *OrganisationsHandler) RegisterHandlers(router *mux.Router) {
	logger.Info("Registering handlers")
	mh := handlers.MethodHandler{
		"GET": http.HandlerFunc(h.GetOrganisation),
	}

	path := "/organisations/{uuid}"
	router.Handle(path, mh)
	router.HandleFunc(path, h.MethodNotAllowedHandler)
}

// HealthCheck does something
func (h *OrganisationsHandler) HealthCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "neo4j-check",
		BusinessImpact:   "Unable to respond to Public Organisations api requests",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://dewey.ft.com/public-org-api.html",
		Severity:         2,
		TechnicalSummary: "Cannot connect to Neo4j a instance with at least one organisation loaded in it",
		Checker:          h.Checker,
	}
}

// Checker does more stuff
func (h *OrganisationsHandler) Checker() (string, error) {
	err := OrganisationDriver.CheckConnectivity()
	if err == nil {
		return "Connectivity to neo4j is ok", err
	}
	return "Error connecting to neo4j", err
}

// Ping says pong
func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

// BuildInfoHandler - This is a stop gap and will be added to when we can define what we should display here
func (h *OrganisationsHandler) BuildInfoHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "build-info")
}

// MethodNotAllowedHandler does stuff
func (h *OrganisationsHandler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

const validUUID = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"
const ontologyPrefix = "http://www.ft.com/ontology"
const organisationOntology = ontologyPrefix + "/organisation/Organisation"

// GetOrganisation is the public API
func (h *OrganisationsHandler) GetOrganisation(w http.ResponseWriter, r *http.Request) {
	uuidMatcher := regexp.MustCompile(validUUID)
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	transID := transactionidutils.GetTransactionIDFromRequest(r)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if uuid == "" || !uuidMatcher.MatchString(uuid) {
		msg := fmt.Sprintf(`uuid '%s' is either missing or invalid`, uuid)
		logger.WithTransactionID(transID).WithUUID(uuid).Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "` + msg + `"}`))
		return
	}

	organisation, found, err := h.getOrganisationViaConceptsAPI(uuid, transID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "failed to return organisation"}`))
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "organisation not found"}`))
		return
	}
	//if the request was not made for the canonical, but an alternate uuid: redirect
	if !strings.Contains(organisation.ID, uuid) {
		validRegexp := regexp.MustCompile(validUUID)
		canonicalUUID := validRegexp.FindString(organisation.ID)
		redirectURL := strings.Replace(r.RequestURI, uuid, canonicalUUID, 1)
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Cache-Control", CacheControlHeader)
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(organisation)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Organisation could not be marshelled, err=` + err.Error() + `"}`))
	}
}

//GoodToGo returns a 503 if the healthcheck fails - suitable for use from varnish to check availability of a node
func (h *OrganisationsHandler) GTG() gtg.Status {
	statusCheck := func() gtg.Status {
		return gtgCheck(h.Checker)
	}
	return gtg.FailFastParallelCheck([]gtg.StatusChecker{statusCheck})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *OrganisationsHandler) getOrganisationViaConceptsAPI(uuid string, transID string) (organisation Organisation, found bool, err error) {
	org := Organisation{}
	reqURL := h.conceptsURL + "/concepts/" + uuid
	request, err := http.NewRequest("GET", reqURL, nil)

	if err != nil {
		msg := fmt.Sprintf("failed to create request to %s", reqURL)
		logger.WithError(err).WithUUID(uuid).WithTransactionID(transID).Error(msg)
		return org, false, err
	}

	request.Header.Set("X-Request-Id", transID)
	resp, err := h.client.Do(request)
	if err != nil {
		msg := fmt.Sprintf("request to %s returned status: %d", reqURL, resp.StatusCode)
		logger.WithError(err).WithUUID(uuid).WithTransactionID(transID).Error(msg)
		return org, false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return org, false, nil
	}

	conceptsApiResponse := ConceptApiResponse{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read response body: %v", resp.Body)
		logger.WithError(err).WithUUID(uuid).WithTransactionID(transID).Error(msg)
		return org, false, err
	}
	if err = json.Unmarshal(body, &conceptsApiResponse); err != nil {
		msg := fmt.Sprintf("failed to unmarshal response body: %v", body)
		logger.WithError(err).WithUUID(uuid).WithTransactionID(transID).Error(msg)
		return org, false, err
	}

	if conceptsApiResponse.Type != organisationOntology {
		logger.WithTransactionID(transID).WithUUID(uuid).Debug("requested concept is not a organisation")
		return org, false, nil
	}

	org.ID = conceptsApiResponse.ID
	org.APIURL = conceptsApiResponse.ApiURL
	org.PrefLabel = conceptsApiResponse.PrefLabel
	org.Types = mapper.FullTypeHierarchy(conceptsApiResponse.Type)
	org.DirectType = conceptsApiResponse.Type
	org.PostalCode = conceptsApiResponse.PostalCode
	org.CountryCode = conceptsApiResponse.CountryCode
	org.CountryOfIncorporation = conceptsApiResponse.CountryOfIncorporation
	org.LegalEntityIdentifier = conceptsApiResponse.LeiCode
	org.YearFounded = conceptsApiResponse.YearFounded

	formerNames := []string{}
	m := make(map[string]bool)
	uniqLabel := []string{}
	for _, label := range conceptsApiResponse.AlternativeLabels {
		compare := func(expected string) bool {
			return strings.TrimPrefix(label.Type, ontologyPrefix) == expected
		}
		switch {
		case compare("/ProperName"):
			org.ProperName = label.Value
		case compare("/ShortName"):
			org.ShortName = label.Value
		case compare("/HiddenLabel"):
			org.HiddenLabel = label.Value
		case compare("/FormerName"):
			formerNames = append(formerNames, label.Value)
		}

		if !m[label.Value] {
			m[label.Value] = true
			uniqLabel = append(uniqLabel, label.Value)
		}
	}
	if len(formerNames) > 0 {
		org.FormerNames = formerNames
	}
	if len(uniqLabel) > 0 {
		org.Labels = uniqLabel
	}

	var subsidiaries = []Subsidiary{}
	for _, item := range conceptsApiResponse.Related {
		c := item.Concept
		if strings.TrimPrefix(item.Predicate, ontologyPrefix) == "/isParentOrganisationOf" {
			parent := &Parent{}
			parent.ID = c.ID
			parent.APIURL = c.ApiURL
			parent.PrefLabel = c.PrefLabel
			parent.DirectType = c.Type
			parent.Types = mapper.FullTypeHierarchy(c.Type)
			org.Parent = parent
		}
		if strings.TrimPrefix(item.Predicate, ontologyPrefix) == "/hasParentOrganisation" {
			subsidiary := Subsidiary{}
			subsidiary.ID = c.ID
			subsidiary.APIURL = c.ApiURL
			subsidiary.PrefLabel = c.PrefLabel
			subsidiary.DirectType = c.Type
			subsidiary.Types = mapper.FullTypeHierarchy(c.Type)
			subsidiaries = append(subsidiaries, subsidiary)
		}
		if strings.TrimPrefix(item.Predicate, ontologyPrefix) == "/issuedTo" {
			f := &FinancialInstrument{}
			f.ID = c.ID
			f.APIURL = c.ApiURL
			f.PrefLabel = c.PrefLabel
			f.DirectType = c.Type
			f.Types = mapper.FullTypeHierarchy(c.Type)
			f.Figi = c.Figi
			org.FinancialInstrument = f
		}
	}
	if len(subsidiaries) > 0 {
		org.Subsidiaries = subsidiaries
	}

	return org, true, nil
}
