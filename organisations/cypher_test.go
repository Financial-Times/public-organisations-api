package organisations

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/memberships-rw-neo4j/memberships"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/people-rw-neo4j/people"
	"github.com/Financial-Times/roles-rw-neo4j/roles"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

// TODO Add Test cases for more of the mapping functions and perhaps mock out back end (although ? if mocking neoism is of value)

// TestNeoReadStructToPersonMandatoryFields checks that madatory fields are set even if they are empty or nil / null
func TestNeoReadStructToOrganisationMandatoryFields(t *testing.T) {
	expected := `{"id":"http://api.ft.com/things/","apiUrl":"http://api.ft.com/things/","types":null}`
	organisation := neoReadStructToOrganisation(neoReadStruct{}, "prod")
	organisationJSON, err := json.Marshal(organisation)
	assert := assert.New(t)
	assert.NoError(err, "Unable to marshal Organisation to JSON")
	assert.Equal(expected, string(organisationJSON))
}

func TestNeoReadStructToOrganisationEnvIsTest(t *testing.T) {
	expected := `{"id":"http://api.ft.com/things/","apiUrl":"http://test.api.ft.com/things/","types":null}`
	organisation := neoReadStructToOrganisation(neoReadStruct{}, "test")
	organisationJSON, err := json.Marshal(organisation)
	assert := assert.New(t)
	assert.NoError(err, "Unable to marshal Organisation to JSON")
	assert.Equal(expected, string(organisationJSON))
}

func TestNeoReadStructToOrganisationMultipleMemberships(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)
	peopleRW, organisationRW, membershipsRW, rolesRW := getServices(t, assert, db, &batchRunner)

	writeBigOrg(assert, peopleRW, organisationRW, membershipsRW, rolesRW)

	defer cleanDB(db, t, assert)
	defer deleteAllViaService(assert, peopleRW, organisationRW, membershipsRW, rolesRW)

	undertest := NewCypherDriver(db, "prod")
	org, found, err := undertest.Read("3e844449-b27f-40d4-b696-2ce9b6137133")
	assert.NoError(err)
	assert.True(found)
	assert.NotNil(org)

	assert.Equal("http://api.ft.com/things/3e844449-b27f-40d4-b696-2ce9b6137133", org.ID)
	assert.Equal("http://api.ft.com/organisations/3e844449-b27f-40d4-b696-2ce9b6137133", org.APIURL)
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/organisation/Organisation")
	assert.Equal("Super, Inc.", org.PrefLabel)

	subsidiary := Subsidiary{}
	subsidiary.Thing = &Thing{}
	subsidiary.ID = "http://api.ft.com/things/f21a5cc0-d326-4e62-b84a-d840c2209fee"
	subsidiary.APIURL = "http://api.ft.com/organisations/f21a5cc0-d326-4e62-b84a-d840c2209fee"
	subsidiary.Types = []string{"http://www.ft.com/ontology/organisation/Organisation"}
	subsidiary.PrefLabel = "Awesome, Inc."

	assertSubsidiaries(assert, org.Subsidiaries, subsidiary)
	assert.Equal((*org.Parent).ID, "http://api.ft.com/things/f9694ba7-eab0-4ce0-8e01-ff64bccb813c")

	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/organisation/Organisation")
	assertListContainsAll(assert, *org.Labels, "Super", "Super Incorporated", "Super, Inc.", "Super Inc.", "Super Inc")
	assertListContainsAll(assert, org.Memberships,
		getMembership(getDan(), "Controller of Awesomeness", ChangeEvent{StartedAt: "2010-12-11T00:00:00Z"}, ChangeEvent{EndedAt: "2012-01-01T00:00:00Z"}),
		getMembership(getNicky(), "Controller of Awesomeness", ChangeEvent{StartedAt: "2009-12-11T00:00:00Z"}, ChangeEvent{EndedAt: "2012-05-01T00:00:00Z"}),
		getMembership(getNicky(), "Party Cat Coordinator", ChangeEvent{StartedAt: "2012-06-01T00:00:00Z"}),
		getMembership(getScott(), "Head of Latin American Research & Strategy"),
		getMembership(getGalia(), "Madame le Président", ChangeEvent{EndedAt: "2012-05-05T00:00:00Z"}))
}

func assertSubsidiaries(assert *assert.Assertions, actual []Subsidiary, items ...Subsidiary) {
	assert.Len(actual, len(items))
	for _, item := range items {
		assert.Contains(actual, item)
	}
}

func assertListContainsAll(assert *assert.Assertions, list interface{}, items ...interface{}) {
	assert.Len(list, len(items))
	for _, item := range items {
		assert.Contains(list, item)
	}
}

func writeBigOrg(assert *assert.Assertions, peopleRW baseftrwapp.Service, organisationRW baseftrwapp.Service, membershipsRW baseftrwapp.Service, rolesRW baseftrwapp.Service) {
	writeJsonToService(peopleRW, "./fixtures/Person-Dan_Murphy-868c3c17-611c-4943-9499-600ccded71f3.json", assert)
	writeJsonToService(peopleRW, "./fixtures/Person-Nicky_Wrightson-fa2ae871-ef77-49c8-a030-8d90eae6cf18.json", assert)
	writeJsonToService(peopleRW, "./fixtures/Person-Scott_Newton-84cec0e1-a866-47bd-9444-d74873b69786.json", assert)
	writeJsonToService(peopleRW, "./fixtures/Person-Galia_Rimon-bdacd96e-d2f4-429f-bb61-462e40448409.json", assert)

	writeJsonToService(organisationRW, "./fixtures/Organisation-Child-f21a5cc0-d326-4e62-b84a-d840c2209fee.json", assert)
	writeJsonToService(organisationRW, "./fixtures/Organisation-Main-3e844449-b27f-40d4-b696-2ce9b6137133.json", assert)
	writeJsonToService(organisationRW, "./fixtures/Organisation-Parent-f9694ba7-eab0-4ce0-8e01-ff64bccb813c.json", assert)

	writeJsonToService(membershipsRW, "./fixtures/Membership-Dan_Murphy-6b278d36-5b30-46a3-b036-55902a9d31ac.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-668c103f-d8dc-4938-9324-9c60de726705.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-c739b972-f41d-43d2-b8d9-5848c92e17f6.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Scott_Newton-177de04f-c09a-4d66-ab55-bb68496c9c28.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Galia_Rimon-9c50e77a-de8a-4f8c-b1dd-09c7730e2c70.json", assert)

	writeJsonToService(rolesRW, "./fixtures/Role-Board-ff9e35f2-63e4-487a-87a4-d82535e047de.json", assert)
	writeJsonToService(rolesRW, "./fixtures/Role-c7063a20-5ca5-4f7a-8a96-47e946b5739e.json", assert)
	writeJsonToService(rolesRW, "./fixtures/Role-d8bbba91-8a87-4dee-bd1a-f79e8139e5c9.json", assert)
	writeJsonToService(rolesRW, "./fixtures/Role-5fcfec9c-8ff0-4ee2-9e91-f270492d636c.json", assert)
}

func deleteAllViaService(assert *assert.Assertions, peopleRW baseftrwapp.Service, organisationRW baseftrwapp.Service, membershipsRW baseftrwapp.Service, rolesRW baseftrwapp.Service) {
	peopleRW.Delete("868c3c17-611c-4943-9499-600ccded71f3")
	peopleRW.Delete("fa2ae871-ef77-49c8-a030-8d90eae6cf18")
	peopleRW.Delete("84cec0e1-a866-47bd-9444-d74873b69786")
	peopleRW.Delete("bdacd96e-d2f4-429f-bb61-462e40448409")

	organisationRW.Delete("f21a5cc0-d326-4e62-b84a-d840c2209fee")
	organisationRW.Delete("3e844449-b27f-40d4-b696-2ce9b6137133")
	organisationRW.Delete("f9694ba7-eab0-4ce0-8e01-ff64bccb813c")

	membershipsRW.Delete("6b278d36-5b30-46a3-b036-55902a9d31ac")
	membershipsRW.Delete("668c103f-d8dc-4938-9324-9c60de726705")
	membershipsRW.Delete("c739b972-f41d-43d2-b8d9-5848c92e17f6")
	membershipsRW.Delete("177de04f-c09a-4d66-ab55-bb68496c9c28")
	membershipsRW.Delete("9c50e77a-de8a-4f8c-b1dd-09c7730e2c70")

	rolesRW.Delete("ff9e35f2-63e4-487a-87a4-d82535e047de")
	rolesRW.Delete("c7063a20-5ca5-4f7a-8a96-47e946b5739e")
	rolesRW.Delete("d8bbba91-8a87-4dee-bd1a-f79e8139e5c9")
	rolesRW.Delete("5fcfec9c-8ff0-4ee2-9e91-f270492d636c")
}

func getServices(t *testing.T, assert *assert.Assertions, db *neoism.Database, batchRunner *neoutils.CypherRunner) (baseftrwapp.Service, baseftrwapp.Service, baseftrwapp.Service, baseftrwapp.Service) {
	peopleRW := people.NewCypherPeopleService(*batchRunner, db)
	assert.NoError(peopleRW.Initialise())
	organisationRW := organisations.NewCypherOrganisationService(*batchRunner, db)
	assert.NoError(organisationRW.Initialise())
	membershipsRW := memberships.NewCypherDriver(*batchRunner, db)
	assert.NoError(membershipsRW.Initialise())
	rolesRW := roles.NewCypherDriver(*batchRunner, db)
	assert.NoError(rolesRW.Initialise())
	return peopleRW, organisationRW, membershipsRW, rolesRW
}

func writeJsonToService(service baseftrwapp.Service, pathToJsonFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJsonFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) *neoism.Database {
	db := getDatabaseConnection(t, assert)
	cleanDB(db, t, assert)
	//	checkDbClean(db, t)
	return db
}

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) *neoism.Database {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db *neoism.Database, t *testing.T, assert *assert.Assertions) {
	uuids := []string{
		"3e844449-b27f-40d4-b696-2ce9b6137133",
		"f21a5cc0-d326-4e62-b84a-d840c2209fee",
		"f9694ba7-eab0-4ce0-8e01-ff64bccb813c",
		"84cec0e1-a866-47bd-9444-d74873b69786",
		"fa2ae871-ef77-49c8-a030-8d90eae6cf18",
		"868c3c17-611c-4943-9499-600ccded71f3",
		"d8bbba91-8a87-4dee-bd1a-f79e8139e5c9",
		"c7063a20-5ca5-4f7a-8a96-47e946b5739e",
		"ff9e35f2-63e4-487a-87a4-d82535e047de",
		"177de04f-c09a-4d66-ab55-bb68496c9c28",
		"6b278d36-5b30-46a3-b036-55902a9d31ac",
		"c739b972-f41d-43d2-b8d9-5848c92e17f6",
		"668c103f-d8dc-4938-9324-9c60de726705",
		"f21a5cc0-d326-4e62-b84a-d840c2209fee",
		"bdacd96e-d2f4-429f-bb61-462e40448409",
		"9c50e77a-de8a-4f8c-b1dd-09c7730e2c70",
		"5fcfec9c-8ff0-4ee2-9e91-f270492d636c",
	}

	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%s'}) DETACH DELETE a", uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}
func getMembership(person *Person, title string, changeEvents ...ChangeEvent) Membership {
	membership := Membership{}
	membership.Title = title
	membership.Person = (*person)
	if len(changeEvents) > 0 {
		membership.ChangeEvents = &changeEvents
	}
	return membership
}

func getDan() *Person {
	person := &Person{}
	person.Thing = &Thing{}
	person.ID = "http://api.ft.com/things/868c3c17-611c-4943-9499-600ccded71f3"
	person.APIURL = "http://api.ft.com/people/868c3c17-611c-4943-9499-600ccded71f3"
	person.Types = []string{"http://www.ft.com/ontology/person/Person"}
	person.PrefLabel = "Dan Murphy"
	return person
}

func getScott() *Person {
	person := &Person{}
	person.Thing = &Thing{}
	person.ID = "http://api.ft.com/things/84cec0e1-a866-47bd-9444-d74873b69786"
	person.APIURL = "http://api.ft.com/people/84cec0e1-a866-47bd-9444-d74873b69786"
	person.Types = []string{"http://www.ft.com/ontology/person/Person"}
	person.PrefLabel = "Scott Newton"
	return person
}

func getNicky() *Person {
	person := &Person{}
	person.Thing = &Thing{}
	person.ID = "http://api.ft.com/things/fa2ae871-ef77-49c8-a030-8d90eae6cf18"
	person.APIURL = "http://api.ft.com/people/fa2ae871-ef77-49c8-a030-8d90eae6cf18"
	person.Types = []string{"http://www.ft.com/ontology/person/Person"}
	person.PrefLabel = "Nicky Wrightson"
	return person
}

func getGalia() *Person {
	person := &Person{}
	person.Thing = &Thing{}
	person.ID = "http://api.ft.com/things/bdacd96e-d2f4-429f-bb61-462e40448409"
	person.APIURL = "http://api.ft.com/people/bdacd96e-d2f4-429f-bb61-462e40448409"
	person.Types = []string{"http://www.ft.com/ontology/person/Person"}
	person.PrefLabel = "Galia Rimon"
	return person
}
