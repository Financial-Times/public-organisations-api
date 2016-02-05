package organisations

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/base-ft-rw-app-go"
	"github.com/Financial-Times/memberships-rw-neo4j/memberships"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/Financial-Times/organisations-rw-neo4j/organisations"
	"github.com/Financial-Times/people-rw-neo4j/people"
	"github.com/Financial-Times/roles-rw-neo4j/roles"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TODO Add Test cases for more of the mapping functions and perhaps mock out back end (although ? if mocking neoism is of value)

// TestNeoReadStructToPersonMandatoryFields checks that madatory fields are set even if they are empty or nil / null
func TestNeoReadStructToOrganisationMandatoryFields(t *testing.T) {
	t.SkipNow()
	// Todo implement
}

func TestNeoReadStructToOrganisationMultipleMemberships(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	batchRunner := neoutils.NewBatchCypherRunner(neoutils.StringerDb{db}, 1)
	peopleRW, organisationRW, membershipsRW, rolesRW := getServices(t, assert, db, &batchRunner)

	writeBigOrg(assert, peopleRW, organisationRW, membershipsRW, rolesRW)

	defer cleanDB(db, t, assert)
	defer deleteAllViaService(assert, peopleRW, organisationRW, membershipsRW, rolesRW)

	undertest := NewCypherDriver(db)
	org, found, err := undertest.Read("3e844449-b27f-40d4-b696-2ce9b6137133")
	assert.NoError(err)
	assert.True(found)
	assert.NotNil(org)

	assert.Equal("http://api.ft.com/things/3e844449-b27f-40d4-b696-2ce9b6137133", org.ID)
	assert.Equal("http://api.ft.com/organisations/3e844449-b27f-40d4-b696-2ce9b6137133", org.APIURL)
	assert.Equal("Super, Inc.", org.PrefLabel)
//	assert.Len(org.Subsidiaries, 1)
//	assert.Equal((*org.Parent).ID, "f9694ba7-eab0-4ce0-8e01-ff64bccb813c")
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/organisation/Organisation")
	assertListContainsAll(assert, *org.Labels, "Super", "Super Incorporated", "Super, Inc.", "Super Inc.", "Super Inc")
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

	writeJsonToService(organisationRW, "./fixtures/Organisation-Child-f21a5cc0-d326-4e62-b84a-d840c2209fee.json", assert)
	writeJsonToService(organisationRW, "./fixtures/Organisation-Main-3e844449-b27f-40d4-b696-2ce9b6137133.json", assert)
	writeJsonToService(organisationRW, "./fixtures/Organisation-Parent-f9694ba7-eab0-4ce0-8e01-ff64bccb813c.json", assert)

	writeJsonToService(membershipsRW, "./fixtures/Membership-Dan_Murphy-6b278d36-5b30-46a3-b036-55902a9d31ac.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-668c103f-d8dc-4938-9324-9c60de726705.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-c739b972-f41d-43d2-b8d9-5848c92e17f6.json", assert)
	writeJsonToService(membershipsRW, "./fixtures/Membership-Scott_Newton-177de04f-c09a-4d66-ab55-bb68496c9c28.json", assert)

	writeJsonToService(rolesRW, "./fixtures/Role-Board-ff9e35f2-63e4-487a-87a4-d82535e047de.json", assert)
	writeJsonToService(rolesRW, "./fixtures/Role-c7063a20-5ca5-4f7a-8a96-47e946b5739e.json", assert)
	writeJsonToService(rolesRW, "./fixtures/Role-d8bbba91-8a87-4dee-bd1a-f79e8139e5c9.json", assert)
}

func deleteAllViaService(assert *assert.Assertions, peopleRW baseftrwapp.Service, organisationRW baseftrwapp.Service, membershipsRW baseftrwapp.Service, rolesRW baseftrwapp.Service) {
	peopleRW.Delete("868c3c17-611c-4943-9499-600ccded71f3")
	peopleRW.Delete("fa2ae871-ef77-49c8-a030-8d90eae6cf18")
	peopleRW.Delete("84cec0e1-a866-47bd-9444-d74873b69786")

	organisationRW.Delete("f21a5cc0-d326-4e62-b84a-d840c2209fee")
	organisationRW.Delete("3e844449-b27f-40d4-b696-2ce9b6137133")
	organisationRW.Delete("f9694ba7-eab0-4ce0-8e01-ff64bccb813c")

	membershipsRW.Delete("6b278d36-5b30-46a3-b036-55902a9d31ac")
	membershipsRW.Delete("668c103f-d8dc-4938-9324-9c60de726705")
	membershipsRW.Delete("c739b972-f41d-43d2-b8d9-5848c92e17f6")
	membershipsRW.Delete("177de04f-c09a-4d66-ab55-bb68496c9c28")

	rolesRW.Delete("ff9e35f2-63e4-487a-87a4-d82535e047de")
	rolesRW.Delete("c7063a20-5ca5-4f7a-8a96-47e946b5739e")
	rolesRW.Delete("d8bbba91-8a87-4dee-bd1a-f79e8139e5c9")
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
	}

	qs := make([]*neoism.CypherQuery, len(uuids))
	for i, uuid := range uuids {
		qs[i] = &neoism.CypherQuery{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%s'}) DETACH DELETE a", uuid)}
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}
