package organisations

import (
	"errors"
	"fmt"
	"time"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	log "github.com/Sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

// Driver interface
type Driver interface {
	Read(id string) (organisation Organisation, found bool, err error)
	CheckConnectivity() error
}

// CypherDriver struct
type CypherDriver struct {
	db  *neoism.Database
	env string
}

//NewCypherDriver instantiate driver
func NewCypherDriver(db *neoism.Database, env string) CypherDriver {
	return CypherDriver{db, env}
}

// CheckConnectivity tests neo4j by running a simple cypher query
func (pcw CypherDriver) CheckConnectivity() error {
	results := []struct {
		ID int
	}{}
	query := &neoism.CypherQuery{
		Statement: "MATCH (x) RETURN ID(x) LIMIT 1",
		Result:    &results,
	}
	err := pcw.db.Cypher(query)
	log.Debugf("CheckConnectivity results:%+v  err: %+v", results, err)
	return err
}

type neoChangeEvent struct {
	StartedAt string
	EndedAt   string
}

type neoReadStruct struct {
	O struct {
		ID        string
		Types     []string
		PrefLabel string
		Labels    []string
	}
	Lei struct {
		LegalEntityIdentifier string
	}
	Parent struct {
		ID        string
		Types     []string
		PrefLabel string
	}
	Ind struct {
		ID        string
		Types     []string
		PrefLabel string
	}
	Sub []struct {
		ID        string
		Types     []string
		PrefLabel string
	}
	PM []struct {
		M struct {
			ID           string
			Types        []string
			PrefLabel    string
			Title        string
			ChangeEvents []neoChangeEvent
		}
		P struct {
			ID        string
			Types     []string
			PrefLabel string
			Labels    []string
		}
	}
	Fi []struct {
		ID 	  string
		PrefLabel string
		Types     []string
		FIGI      string
	}
}

func (pcw CypherDriver) Read(uuid string) (organisation Organisation, found bool, err error) {
	organisation = Organisation{}
	results := []struct {
		Rs []neoReadStruct
	}{}
	query := &neoism.CypherQuery{
		Statement: `
		MATCH (identifier:UPPIdentifier{value:{uuid}})
		MATCH (identifier)-[:IDENTIFIES]->(o:Organisation)
 		OPTIONAL MATCH (o)<-[:HAS_ORGANISATION]-(m:Membership)
 		OPTIONAL MATCH (m)-[:HAS_MEMBER]->(p:Person)
 		OPTIONAL MATCH (p)<-[:MENTIONS]-(poc:Content)-[:MENTIONS]->(o)
 		WITH    o,
 		{ id:p.uuid, types:labels(p), prefLabel:p.prefLabel} as p,
 		{ id:m.uuid, prefLabel:m.prefLabel, changeEvents:[{startedAt:m.inceptionDate}, {endedAt:m.terminationDate}], annCount:COUNT(poc) } as m ORDER BY m.annCount DESC LIMIT 1000
 		WITH o, collect({m:m, p:p}) as pm
 		OPTIONAL MATCH (o)-[:HAS_CLASSIFICATION]->(ind:IndustryClassification)
 		WITH o, pm,
 		{ id:ind.uuid, types:labels(ind), prefLabel:ind.prefLabel} as ind
 		WITH o, pm, ind
 		OPTIONAL MATCH (lei:LegalEntityIdentifier)-[:IDENTIFIES]->(o)
		WITH o, pm, ind,
 		{ legalEntityIdentifier:lei.value } as lei
		WITH o, pm, ind, lei
 		OPTIONAL MATCH (o)-[:SUB_ORGANISATION_OF]->(parent:Organisation)
 		OPTIONAL MATCH (o)<-[:SUB_ORGANISATION_OF]-(sub:Organisation)
 		OPTIONAL MATCH (soc:Content)-[ms:MENTIONS]->(sub)
 		WITH o, pm, ind, lei,
 		{ id:parent.uuid, types:labels(parent), prefLabel:parent.prefLabel} as parent,
 		{ id:sub.uuid, types:labels(sub), prefLabel:sub.prefLabel, annCount:COUNT(ms) } as sub ORDER BY sub.annCount DESC
 		WITH o, pm, ind, lei, parent, collect(sub) as sub
 		OPTIONAL MATCH (fi:FinancialInstrument)-[:ISSUED_BY]->(o)
 		OPTIONAL MATCH (fi)<-[:IDENTIFIES]-(figi:FIGIIdentifier)
 		WITH o, pm, ind, lei, parent, sub,
 		{id:fi.uuid, types:labels(fi), prefLabel:fi.prefLabel, figi:figi.value} as fi
		WITH pm, ind, parent, sub, lei, collect(fi) as fi, { id:o.uuid, types:labels(o), prefLabel:o.prefLabel, labels:o.aliases} as o
 		return collect ({o:o, lei:lei, parent:parent, ind:ind, sub:sub, pm:pm, fi:fi}) as rs
							`,
		Parameters: neoism.Props{"uuid": uuid},
		Result:     &results,
	}
	err = pcw.db.Cypher(query)
	if err != nil {
		log.Errorf("Error looking up uuid %s with query %s from neoism: %+v\n", uuid, query.Statement, err)
		return Organisation{}, false, fmt.Errorf("Error accessing Organisation datastore for uuid: %s", uuid)
	}
	log.Debugf("CypherResult ReadOrganisation for uuid: %s was: %+v", uuid, results)
	if (len(results)) == 0 || len(results[0].Rs) == 0 {
		return Organisation{}, false, nil
	} else if len(results) != 1 && len(results[0].Rs) != 1 {
		errMsg := fmt.Sprintf("Multiple organisations found with the same uuid:%s !", uuid)
		log.Error(errMsg)
		return Organisation{}, true, errors.New(errMsg)
	}
	organisation = neoReadStructToOrganisation(results[0].Rs[0], pcw.env)
	log.Debugf("Returning %v", organisation)
	return organisation, true, nil
}

func neoReadStructToOrganisation(neo neoReadStruct, env string) Organisation {
	//TODO find out why we only get two memberships here compared to 17 off PROD graphDB... also, performance of e.g. Barclays
	public := Organisation{}
	public.Thing = &Thing{}
	public.ID = mapper.IDURL(neo.O.ID)
	public.APIURL = mapper.APIURL(neo.O.ID, neo.O.Types, env)
	public.Types = mapper.TypeURIs(neo.O.Types)
	public.PrefLabel = neo.O.PrefLabel
	if len(neo.O.Labels) > 0 {
		public.Labels = &neo.O.Labels
	}

	if neo.Lei.LegalEntityIdentifier != "" {
		public.LegalEntityIdentifier = neo.Lei.LegalEntityIdentifier
	}

	if neo.Ind.ID != "" {
		public.IndustryClassification = &IndustryClassification{}
		public.IndustryClassification.Thing = &Thing{}
		public.IndustryClassification.ID = mapper.IDURL(neo.Ind.ID)
		public.IndustryClassification.APIURL = mapper.APIURL(neo.Ind.ID, neo.Ind.Types, env)
		public.IndustryClassification.PrefLabel = neo.Ind.PrefLabel
	}

	if len(neo.Fi) == 1 && neo.Fi[0].ID == "" {
		public.FinancialInstruments = make([]FinancialInstrument, 0, 0)
	} else {
		public.FinancialInstruments = make([]FinancialInstrument, len(neo.Fi))
		for idx, fi := range neo.Fi {
			financialInstrument := FinancialInstrument{}
			financialInstrument.Thing = &Thing{}
			financialInstrument.ID = mapper.IDURL(fi.ID)
			financialInstrument.APIURL = mapper.APIURL(fi.ID, fi.Types, env)
			financialInstrument.Types = mapper.TypeURIs(fi.Types)
			financialInstrument.PrefLabel = fi.PrefLabel
			financialInstrument.Figi = fi.FIGI
			public.FinancialInstruments[idx] = financialInstrument
		}
	}

	if neo.Parent.ID != "" {
		public.Parent = &Parent{}
		public.Parent.Thing = &Thing{}
		public.Parent.ID = mapper.IDURL(neo.Parent.ID)
		public.Parent.APIURL = mapper.APIURL(neo.Parent.ID, neo.Parent.Types, env)
		public.Parent.Types = mapper.TypeURIs(neo.Parent.Types)
		public.Parent.PrefLabel = neo.Parent.PrefLabel
	}

	if len(neo.Sub) == 1 && neo.Sub[0].ID == "" {
		public.Subsidiaries = make([]Subsidiary, 0, 0)
	} else {
		public.Subsidiaries = make([]Subsidiary, len(neo.Sub))
		for idx, neoSub := range neo.Sub {
			subsidiary := Subsidiary{}
			subsidiary.Thing = &Thing{}
			subsidiary.ID = mapper.IDURL(neoSub.ID)
			subsidiary.APIURL = mapper.APIURL(neoSub.ID, neoSub.Types, env)
			subsidiary.Types = mapper.TypeURIs(neoSub.Types)
			subsidiary.PrefLabel = neoSub.PrefLabel
			public.Subsidiaries[idx] = subsidiary
		}
	}

	if len(neo.PM) == 1 && (neo.PM[0].M.ID == "") {
		public.Memberships = make([]Membership, 0, 0)
	} else {
		public.Memberships = make([]Membership, len(neo.PM))
		for mIdx, neoMem := range neo.PM {
			membership := Membership{}
			membership.Title = neoMem.M.PrefLabel
			membership.Person = Person{}
			membership.Person.Thing = &Thing{}
			membership.Person.ID = mapper.IDURL(neoMem.P.ID)
			membership.Person.APIURL = mapper.APIURL(neoMem.P.ID, neoMem.P.Types, env)
			membership.Person.Types = mapper.TypeURIs(neoMem.P.Types)
			membership.Person.PrefLabel = neoMem.P.PrefLabel
			if a, b := changeEvent(neoMem.M.ChangeEvents); a == true {
				membership.ChangeEvents = b
			}
			public.Memberships[mIdx] = membership
		}
	}
	log.Debugf("neoReadStructToOrganisation neo: %+v result: %+v", neo, public)
	return public
}

func changeEvent(neoChgEvts []neoChangeEvent) (bool, *[]ChangeEvent) {
	var results []ChangeEvent
	currentLayout := "2006-01-02T15:04:05.999Z"
	layout := "2006-01-02T15:04:05Z"

	if neoChgEvts[0].StartedAt == "" && neoChgEvts[1].EndedAt == "" {
		results = make([]ChangeEvent, 0, 0)
		return false, &results
	}
	for _, neoChgEvt := range neoChgEvts {
		if neoChgEvt.StartedAt != "" {
			t, _ := time.Parse(currentLayout, neoChgEvt.StartedAt)
			results = append(results, ChangeEvent{StartedAt: t.Format(layout)})
		}
		if neoChgEvt.EndedAt != "" {
			t, _ := time.Parse(layout, neoChgEvt.EndedAt)
			results = append(results, ChangeEvent{EndedAt: t.Format(layout)})
		}
	}
	log.Debugf("changeEvent converted: %+v result:%+v", neoChgEvts, results)
	return true, &results
}
