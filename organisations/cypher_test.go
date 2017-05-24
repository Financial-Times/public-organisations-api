package organisations

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/financial-instruments-rw-neo4j/financialinstruments"
	"github.com/Financial-Times/memberships-rw-neo4j/memberships"
	"github.com/Financial-Times/neo-utils-go/neoutils"
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

func TestNeoReadOrganisationWithCanonicalUPPID(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	_, organisationRW, _, _, _ := getServices(t, assert, db)

	writeOrg(assert, organisationRW, "./fixtures/Organisation-Complex-f21a5cc0-d326-4e62-b84a-d840c2209fee.json")

	defer cleanDB(db, t, assert)
	defer deleteOrgViaService(assert, organisationRW, "f21a5cc0-d326-4e62-b84a-d840c2209fee")

	undertest := NewCypherDriver(db, "prod")
	org, found, err := undertest.Read("f21a5cc0-d326-4e62-b84a-d840c2209fee")
	assert.NoError(err)
	assert.True(found)
	assert.NotNil(org)

	assert.Equal("http://api.ft.com/things/f21a5cc0-d326-4e62-b84a-d840c2209fee", org.ID)
	assert.Equal("http://api.ft.com/organisations/f21a5cc0-d326-4e62-b84a-d840c2209fee", org.APIURL)
	assert.Equal("7ZW8QJWVPR4P1J1KQY46", org.LegalEntityIdentifier)
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation")
	assert.Equal("Awesome, Inc.", org.PrefLabel)

}

func TestNeoReadOrganisationWithAlternateUPPID(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	_, organisationRW, _, _, _ := getServices(t, assert, db)

	writeOrg(assert, organisationRW, "./fixtures/Organisation-Complex-f21a5cc0-d326-4e62-b84a-d840c2209fee.json")

	canonicalUUID := "f21a5cc0-d326-4e62-b84a-d840c2209fee"
	alternateUUID := "2421f3fa-5a6a-4320-88f5-7926d1cb2379"

	defer cleanDB(db, t, assert)
	defer deleteOrgViaService(assert, organisationRW, canonicalUUID)

	undertest := NewCypherDriver(db, "prod")
	org, found, err := undertest.Read(alternateUUID)
	assert.NoError(err)
	assert.True(found)
	assert.NotNil(org)

	assert.Equal("http://api.ft.com/things/"+canonicalUUID, org.ID)
	assert.Equal("http://api.ft.com/organisations/"+canonicalUUID, org.APIURL)
	assert.Equal("7ZW8QJWVPR4P1J1KQY46", org.LegalEntityIdentifier)
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation")
	assert.Equal("Awesome, Inc.", org.PrefLabel)

}

func TestNeoReadOrganisationWithMissingUPPIDShouldReturnEmptyOrg(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	_, organisationRW, _, _, _ := getServices(t, assert, db)

	writeOrg(assert, organisationRW, "./fixtures/Organisation-Complex-f21a5cc0-d326-4e62-b84a-d840c2209fee.json")

	canonicalUUID := "f21a5cc0-d326-4e62-b84a-d840c2209fee"

	defer cleanDB(db, t, assert)
	defer deleteOrgViaService(assert, organisationRW, canonicalUUID)

	removeUppId := neoism.CypherQuery{
		Statement: fmt.Sprintf("MATCH (upp:UPPIdentifier)-[:IDENTIFIES]->(o:Organisation{uuid:'%v'}) DETACH DELETE upp", canonicalUUID),
	}

	assert.NoError(db.CypherBatch([]*neoism.CypherQuery{&removeUppId}))

	undertest := NewCypherDriver(db, "prod")
	org, found, err := undertest.Read(canonicalUUID)
	assert.NoError(err)
	assert.False(found)
	assert.NotNil(org)
	assert.Equal(Organisation{}, org)
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
	peopleRW, organisationRW, membershipsRW, rolesRW, _ := getServices(t, assert, db)

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
	assert.Equal("7ZW8QJWVPR4P1J1KQY45", org.LegalEntityIdentifier)
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation")
	assert.Equal("Super, Inc.", org.PrefLabel)

	subsidiary := Subsidiary{}
	subsidiary.Thing = &Thing{}
	subsidiary.ID = "http://api.ft.com/things/f21a5cc0-d326-4e62-b84a-d840c2209fee"
	subsidiary.APIURL = "http://api.ft.com/organisations/f21a5cc0-d326-4e62-b84a-d840c2209fee"
	subsidiary.Types = []string{"http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation"}
	subsidiary.PrefLabel = "Awesome, Inc."

	assertSubsidiaries(assert, org.Subsidiaries, subsidiary)
	assert.Equal((*org.Parent).ID, "http://api.ft.com/things/f9694ba7-eab0-4ce0-8e01-ff64bccb813c")

	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation")
	assertListContainsAll(assert, *org.Labels, "Super", "Super Incorporated", "Super, Inc.", "Super Inc.", "Super Inc")
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

func writeOrg(assert *assert.Assertions, organisationRW baseftrwapp.Service, path string) {
	writeJSONToService(organisationRW, path, assert)
}

func writeFinancialInstrument(assert *assert.Assertions, financialInstrumentsRW baseftrwapp.Service, path string) {
	writeJSONToService(financialInstrumentsRW, path, assert)
}

func writeBigOrg(assert *assert.Assertions, peopleRW baseftrwapp.Service, organisationRW baseftrwapp.Service, membershipsRW baseftrwapp.Service, rolesRW baseftrwapp.Service) {
	writeJSONToService(peopleRW, "./fixtures/Person-Dan_Murphy-868c3c17-611c-4943-9499-600ccded71f3.json", assert)
	writeJSONToService(peopleRW, "./fixtures/Person-Nicky_Wrightson-fa2ae871-ef77-49c8-a030-8d90eae6cf18.json", assert)
	writeJSONToService(peopleRW, "./fixtures/Person-Scott_Newton-84cec0e1-a866-47bd-9444-d74873b69786.json", assert)
	writeJSONToService(peopleRW, "./fixtures/Person-Galia_Rimon-bdacd96e-d2f4-429f-bb61-462e40448409.json", assert)

	writeJSONToService(organisationRW, "./fixtures/Organisation-Child-f21a5cc0-d326-4e62-b84a-d840c2209fee.json", assert)
	writeJSONToService(organisationRW, "./fixtures/Organisation-Main-3e844449-b27f-40d4-b696-2ce9b6137133.json", assert)
	writeJSONToService(organisationRW, "./fixtures/Organisation-Parent-f9694ba7-eab0-4ce0-8e01-ff64bccb813c.json", assert)

	writeJSONToService(membershipsRW, "./fixtures/Membership-Dan_Murphy-6b278d36-5b30-46a3-b036-55902a9d31ac.json", assert)
	writeJSONToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-668c103f-d8dc-4938-9324-9c60de726705.json", assert)
	writeJSONToService(membershipsRW, "./fixtures/Membership-Nicky_Wrightson-c739b972-f41d-43d2-b8d9-5848c92e17f6.json", assert)
	writeJSONToService(membershipsRW, "./fixtures/Membership-Scott_Newton-177de04f-c09a-4d66-ab55-bb68496c9c28.json", assert)
	writeJSONToService(membershipsRW, "./fixtures/Membership-Galia_Rimon-9c50e77a-de8a-4f8c-b1dd-09c7730e2c70.json", assert)

	writeJSONToService(rolesRW, "./fixtures/Role-Board-ff9e35f2-63e4-487a-87a4-d82535e047de.json", assert)
	writeJSONToService(rolesRW, "./fixtures/Role-c7063a20-5ca5-4f7a-8a96-47e946b5739e.json", assert)
	writeJSONToService(rolesRW, "./fixtures/Role-d8bbba91-8a87-4dee-bd1a-f79e8139e5c9.json", assert)
	writeJSONToService(rolesRW, "./fixtures/Role-5fcfec9c-8ff0-4ee2-9e91-f270492d636c.json", assert)
}

func deleteOrgViaService(assert *assert.Assertions, organisationRW baseftrwapp.Service, uuid string) {
	organisationRW.Delete(uuid)
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

func deleteFinancialInstrumentViaService(assert *assert.Assertions, financialInstrument baseftrwapp.Service, uuid string) {
	_, err := financialInstrument.Delete(uuid)
	assert.NoError(err)
}

func getServices(t *testing.T, assert *assert.Assertions, db neoutils.NeoConnection) (baseftrwapp.Service, baseftrwapp.Service, baseftrwapp.Service, baseftrwapp.Service, baseftrwapp.Service) {
	peopleRW := people.NewCypherPeopleService(db)
	assert.NoError(peopleRW.Initialise())
	organisationRW := organisations.NewCypherOrganisationService(db)
	assert.NoError(organisationRW.Initialise())
	membershipsRW := memberships.NewCypherMembershipService(db)
	assert.NoError(membershipsRW.Initialise())
	rolesRW := roles.NewCypherDriver(db)
	assert.NoError(rolesRW.Initialise())
	financialInstrumentsRW := financialinstruments.NewCypherFinancialInstrumentService(db)
	assert.NoError(financialInstrumentsRW.Initialise())
	return peopleRW, organisationRW, membershipsRW, rolesRW, financialInstrumentsRW
}

func writeJSONToService(service baseftrwapp.Service, pathToJSONFile string, assert *assert.Assertions) {
	f, err := os.Open(pathToJSONFile)
	assert.NoError(err)
	dec := json.NewDecoder(f)
	inst, _, errr := service.DecodeJSON(dec)
	assert.NoError(errr)
	errrr := service.Write(inst)
	assert.NoError(errrr)
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(t, assert)
	cleanDB(db, t, assert)
	return db
}

func getDatabaseConnection(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func cleanDB(db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "f21a5cc0-d326-4e62-b84a-d840c2209fee"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "84cec0e1-a866-47bd-9444-d74873b69786"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "fa2ae871-ef77-49c8-a030-8d90eae6cf18"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "868c3c17-611c-4943-9499-600ccded71f3"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "d8bbba91-8a87-4dee-bd1a-f79e8139e5c9"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "c7063a20-5ca5-4f7a-8a96-47e946b5739e"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "ff9e35f2-63e4-487a-87a4-d82535e047de"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "177de04f-c09a-4d66-ab55-bb68496c9c28"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "6b278d36-5b30-46a3-b036-55902a9d31ac"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "c739b972-f41d-43d2-b8d9-5848c92e17f6"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "668c103f-d8dc-4938-9324-9c60de726705"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "f21a5cc0-d326-4e62-b84a-d840c2209fee"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "bdacd96e-d2f4-429f-bb61-462e40448409"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "9c50e77a-de8a-4f8c-b1dd-09c7730e2c70"),
		},
		{
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "5fcfec9c-8ff0-4ee2-9e91-f270492d636c"),
		},
		{
			//deletes parent 'org' which only has type Thing
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "3e844449-b27f-40d4-b696-2ce9b6137133"),
		},
		{
			//deletes parent 'org' which only has type Thing
			Statement: fmt.Sprintf("MATCH (a:Thing {uuid: '%v'}) OPTIONAL MATCH (a)-[]-(b:Identifier) DETACH DELETE a,b", "f9694ba7-eab0-4ce0-8e01-ff64bccb813c"),
		},
	}
	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func TestNeoReadOrganisationWithFinancialInstrument(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	_, organisationRW, _, _, financialInstrumentsRW := getServices(t, assert, db)

	writeOrg(assert, organisationRW, "./fixtures/Organisation-Complex-f21a5cc0-d326-4e62-b84a-d840c2209fee.json")
	writeFinancialInstrument(assert, financialInstrumentsRW, "./fixtures/FinancialInstrument-0c4461d1-0ed3-324f-bbb3-ae948bd3bb09.json")

	defer cleanDB(db, t, assert)
	defer deleteOrgViaService(assert, organisationRW, "f21a5cc0-d326-4e62-b84a-d840c2209fee")
	defer deleteFinancialInstrumentViaService(assert, financialInstrumentsRW, "0c4461d1-0ed3-324f-bbb3-ae948bd3bb09")

	orgService := NewCypherDriver(db, "prod")
	org, found, err := orgService.Read("f21a5cc0-d326-4e62-b84a-d840c2209fee")
	assert.NoError(err)
	assert.True(found)
	assert.NotNil(org.FinancialInstrument)

	assert.Equal("http://api.ft.com/things/f21a5cc0-d326-4e62-b84a-d840c2209fee", org.ID)
	assert.Equal("http://api.ft.com/organisations/f21a5cc0-d326-4e62-b84a-d840c2209fee", org.APIURL)
	assert.Equal("7ZW8QJWVPR4P1J1KQY46", org.LegalEntityIdentifier)
	assertListContainsAll(assert, org.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/organisation/Organisation")
	assert.Equal("Awesome, Inc.", org.PrefLabel)

	assert.Equal("http://api.ft.com/things/0c4461d1-0ed3-324f-bbb3-ae948bd3bb09", org.FinancialInstrument.ID)
	assert.Equal("http://api.ft.com/things/0c4461d1-0ed3-324f-bbb3-ae948bd3bb09", org.FinancialInstrument.APIURL)
	assert.Equal("Emergency Pest Services, Inc.", org.FinancialInstrument.PrefLabel)
	assertListContainsAll(assert, org.FinancialInstrument.Types, "http://www.ft.com/ontology/core/Thing", "http://www.ft.com/ontology/concept/Concept", "http://www.ft.com/ontology/FinancialInstrument")
	assert.Equal("BBG000BQVGX3", org.FinancialInstrument.Figi)
}
