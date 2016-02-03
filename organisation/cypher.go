package organisation

import (
	"errors"
	"fmt"

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
	db *neoism.Database
}

//NewCypherDriver instantiate driver
func NewCypherDriver(db *neoism.Database) CypherDriver {
	return CypherDriver{db}
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
		LEICode   string
		PrefLabel string
		Labels    []string
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
}

func (pcw CypherDriver) Read(uuid string) (organisation Organisation, found bool, err error) {
	organisation = Organisation{}
	results := []struct {
		Rs []neoReadStruct
	}{}
	query := &neoism.CypherQuery{
		Statement: `
				MATCH (o:Organisation{uuid:{uuid}})
				OPTIONAL MATCH (o)<-[:HAS_ORGANISATION]-(m:Membership)
				OPTIONAL MATCH (m)-[:HAS_MEMBER]->(p:Person)
				OPTIONAL MATCH (p)<-[rel:MENTIONS]-(poc:Content)-[mo:MENTIONS]->(o)
				WITH    o,
				{ id:p.uuid, types:labels(p), prefLabel:p.prefLabel} as p,
				{ id:m.uuid, prefLabel:m.prefLabel, changeEvents:[{startedAt:m.inceptionDate}, {endedAt:m.terminationDate}], annCount:COUNT(poc) } as m ORDER BY m.annCount DESC LIMIT 20
				WITH o, collect({m:m, p:p}) as pm
				OPTIONAL MATCH (o)-[:SUB_ORGANISATION_OF]->(parent:Organisation)
				OPTIONAL MATCH (o)<-[:SUB_ORGANISATION_OF]-(sub:Organisation)
				OPTIONAL MATCH (soc:Content)-[mo:MENTIONS]->(sub)
				WITH o, pm,
				{ id:parent.uuid, types:labels(parent), prefLabel:parent.prefLabel} as parent,
				{ id:sub.uuid, types:labels(sub), prefLabel:sub.prefLabel, annCount:COUNT(soc) } as sub ORDER BY sub.annCount DESC LIMIT 10
				WITH o, pm, parent, collect(sub) as subs
				OPTIONAL MATCH (o)-[:HAS_CLASSIFICATION]->(ind:IndustryClassification)
				WITH o, pm, parent, subs,
				{ id:ind.uuid, types:labels(ind), prefLabel:ind.prefLabel} as ind
				WITH pm, parent, subs, ind, { id:o.uuid, types:labels(o), prefLabel:o.prefLabel, labels:o.aliases, leicode:o.leiCode} as o
				return collect ({o:o, ind:ind, parent:parent, subs:subs, pm:pm}) as rs
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
	organisation = neoReadStructToOrganisation(results[0].Rs[0])
	log.Debugf("Returning %v", organisation)
	return organisation, true, nil
}

func neoReadStructToOrganisation(neo neoReadStruct) Organisation {
	//TODO find out why we only get two memberships here compared to 17 off PROD graphDB... also, performance of e.g. Barclays
	public := Organisation{}
	public.Thing = &Thing{}
	public.ID = mapper.IDURL(neo.O.ID)
	public.APIURL = mapper.APIURL(neo.O.ID, neo.O.Types)
	public.Types = mapper.TypeURIs(neo.O.Types)
	public.LEICode = neo.O.LEICode
	public.PrefLabel = neo.O.PrefLabel
	if len(neo.O.Labels) > 0 {
		public.Labels = &neo.O.Labels
	}

	if neo.Ind.ID != "" {
		public.IndustryClassification = &IndustryClassification{}
		public.IndustryClassification.Thing = &Thing{}
		public.IndustryClassification.ID = mapper.IDURL(neo.Ind.ID)
		public.IndustryClassification.APIURL = mapper.APIURL(neo.Ind.ID, neo.Ind.Types)
		public.IndustryClassification.PrefLabel = neo.Ind.PrefLabel
	}
	log.Infof("IndustryClassification=%v", public.IndustryClassification)

	if neo.Parent.ID != "" {
		public.Parent = &Parent{}
		public.Parent.Thing = &Thing{}
		public.Parent.ID = mapper.IDURL(neo.Parent.ID)
		public.Parent.APIURL = mapper.APIURL(neo.Parent.ID, neo.Parent.Types)
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
			subsidiary.APIURL = mapper.APIURL(neoSub.ID, neoSub.Types)
			subsidiary.Types = mapper.TypeURIs(neoSub.Types)
			subsidiary.PrefLabel = neoSub.PrefLabel
			public.Subsidiaries[idx] = subsidiary
		}
	}

	log.Info("LENGTH of memberships:", len(neo.PM))
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
			membership.Person.APIURL = mapper.APIURL(neoMem.P.ID, neoMem.P.Types)
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
	if neoChgEvts[0].StartedAt == "" && neoChgEvts[0].EndedAt == "" {
		results = make([]ChangeEvent, 0, 0)
		return false, &results
	}
	for _, neoChgEvt := range neoChgEvts {
		if neoChgEvt.StartedAt != "" {
			results = append(results, ChangeEvent{StartedAt: neoChgEvt.StartedAt})
		}
		if neoChgEvt.EndedAt != "" {
			results = append(results, ChangeEvent{EndedAt: neoChgEvt.EndedAt})
		}
	}
	log.Debugf("changeEvent converted: %+v result:%+v", neoChgEvts, results)
	return true, &results
}
