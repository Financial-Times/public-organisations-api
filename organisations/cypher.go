package organisations

import (
	"errors"
	"fmt"

	"github.com/Financial-Times/neo-model-utils-go/mapper"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	log "github.com/sirupsen/logrus"
)

// Driver interface
type Driver interface {
	Read(id string) (organisation Organisation, found bool, err error)
	CheckConnectivity() error
}

// CypherDriver struct
type CypherDriver struct {
	conn neoutils.NeoConnection
	env  string
}

//NewCypherDriver instantiate driver
func NewCypherDriver(conn neoutils.NeoConnection, env string) CypherDriver {
	return CypherDriver{conn, env}
}

// CheckConnectivity tests neo4j by running a simple cypher query
func (pcw CypherDriver) CheckConnectivity() error {
	return neoutils.Check(pcw.conn)
}

type neoChangeEvent struct {
	StartedAt string
	EndedAt   string
}

type neoReadStruct struct {
	ID                     string
	Types                  []string
	DirectType             string
	PrefLabel              string
	Labels                 []string
	ProperName             string
	ShortName              string
	HiddenLabel            string
	FormerNames            string
	CountryCode            string
	CountryOfIncorporation string
	PostalCode             string
	YearFounded            int
	Lei                    struct {
		LegalEntityIdentifier string
	}
	Parent struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	Ind struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	Sub []struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	PM []struct {
		M struct {
			ID           string
			Types        []string
			DirectType   string
			PrefLabel    string
			Title        string
			ChangeEvents []neoChangeEvent
		}
		P struct {
			ID         string
			Types      []string
			DirectType string
			PrefLabel  string
			Labels     []string
		}
	}
	Fi struct {
		ID         string
		PrefLabel  string
		Types      []string
		DirectType string
		FIGI       string
	}
}

type neoNewFormatReadStruct struct {
	ID                     string
	Types                  []string
	DirectType             string
	PrefLabel              string
	Labels                 []string
	ProperName             string
	ShortName              string
	HiddenLabel            string
	FormerNames            string
	CountryCode            string
	CountryOfIncorporation string
	PostalCode             string
	YearFounded            int
	Lei                    struct {
		LegalEntityIdentifier string
	}
	Parent []struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	Ind []struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	Sub []struct {
		ID         string
		Types      []string
		DirectType string
		PrefLabel  string
	}
	Fi []struct {
		ID         string
		PrefLabel  string
		Types      []string
		DirectType string
		FIGI       string
	}
}

func (pcw CypherDriver) Read(uuid string) (Organisation, bool, error) {
	org, found, err := pcw.ReadNewFormat(uuid)
	if err != nil {
		return Organisation{}, false, err
	}

	if found {
		return org, found, nil
	}

	return pcw.ReadOldFormat(uuid)
}

func (pcw CypherDriver) ReadNewFormat(uuid string) (organisation Organisation, found bool, err error) {
	organisation = Organisation{}
	results := []struct {
		Rs neoNewFormatReadStruct
	}{}
	query := &neoism.CypherQuery{
		Statement: `
			MATCH (canonical:Thing {prefUUID:"dd128106-3382-406f-8dfb-f4c69dcbbdfb"})<-[:EQUIVALENT_TO]-(source:Thing)
			OPTIONAL MATCH (source)-[:HAS_CLASSIFICATION]->(industryClassification:Thing)
			OPTIONAL MATCH (source)-[:SUB_ORGANISATION_OF]->(parentOrganisation:Thing)
			OPTIONAL MATCH (source)<-[:SUB_ORGANISATION_OF]-(subOrganisation:Thing)
			OPTIONAL MATCH (source)<-[:ISSUED_BY]-(financialInstrument:Thing)
			WITH
				canonical,
				industryClassification,
				parentOrganisation,
				subOrganisation,
				financialInstrument
			WITH
				canonical,
				{
					id: industryClassification.uuid,
					types: labels(industryClassification),
					prefLabel: industryClassification.prefLabel
				} as ind,
				{
					legalEntityIdentifier: canonical.leiCode
				} as lei,
				{
					id: parentOrganisation.uuid,
					types: labels(parentOrganisation),
					prefLabel: parentOrganisation.prefLabel
				} as parent,
				{
					id: financialInstrument.uuid,
					types: labels(financialInstrument),
					prefLabel: financialInstrument.prefLabel,
					figi: financialInstrument.figiCode
				} as fi,
				{
					id: subOrganisation.uuid,
					types: labels(subOrganisation),
					prefLabel: subOrganisation.prefLabel,
					annCount: size((:Content)-[:MENTIONS]->(subOrganisation))
				} as sub
			RETURN
				{
					id: canonical.prefUUID,
					types: labels(canonical),
					prefLabel: canonical.prefLabel,
					labels: canonical.aliases,
					lei: lei,
					properName:canonical.properName,
					shortName: canonical.shortName,
					hiddenLabel: canonical.hiddenLabel,
					formerNames: canonical.formerNames,
					countryCode: canonical.countryCode,
					countryOfIncorporation: canonical.countryOfIncorporation,
					postalCode: canonical.postalCode,
					yearFounded: canonical.yearFounded,
					parent: collect(parent),
					ind: collect(ind),
					fi: collect(fi),
					sub: collect(sub)
				} as rs
		`,
		Parameters: neoism.Props{"uuid": uuid},
		Result:     &results,
	}

	if err := pcw.conn.CypherBatch([]*neoism.CypherQuery{query}); err != nil || len(results) == 0 {
		return Organisation{}, false, err
	}

	complexReadStruct := results[0].Rs
	cleanReadStruct := neoReadStruct{
		ID:                     complexReadStruct.ID,
		Types:                  complexReadStruct.Types,
		DirectType:             complexReadStruct.DirectType,
		PrefLabel:              complexReadStruct.PrefLabel,
		Labels:                 complexReadStruct.Labels,
		Lei:                    complexReadStruct.Lei,
		ProperName:             complexReadStruct.ProperName,
		ShortName:              complexReadStruct.ShortName,
		HiddenLabel:            complexReadStruct.HiddenLabel,
		FormerNames:            complexReadStruct.FormerNames,
		CountryCode:            complexReadStruct.CountryCode,
		CountryOfIncorporation: complexReadStruct.CountryOfIncorporation,
		PostalCode:             complexReadStruct.PostalCode,
		YearFounded:            complexReadStruct.YearFounded,
	}
	for _, parent := range complexReadStruct.Parent {
		if parent.ID == "" {
			continue
		}
		cleanReadStruct.Parent = parent
	}
	for _, ind := range complexReadStruct.Ind {
		if ind.ID == "" {
			continue
		}
		cleanReadStruct.Ind = ind
	}
	for _, sub := range complexReadStruct.Sub {
		if sub.ID == "" {
			continue
		}
		cleanReadStruct.Sub = append(cleanReadStruct.Sub, sub)
	}
	for _, fi := range complexReadStruct.Fi {
		if fi.ID == "" {
			continue
		}
		cleanReadStruct.Fi = fi
	}

	organisation = neoReadStructToOrganisation(cleanReadStruct, pcw.env)
	return organisation, true, nil
}

func (pcw CypherDriver) ReadOldFormat(uuid string) (organisation Organisation, found bool, err error) {
	organisation = Organisation{}
	results := []struct {
		Rs neoReadStruct
	}{}
	query := &neoism.CypherQuery{
		Statement: `
		MATCH (identifier:UPPIdentifier{value:{uuid}})
		MATCH (identifier)-[:IDENTIFIES]->(o:Organisation)
		OPTIONAL MATCH (o)-[:HAS_CLASSIFICATION]->(ind:IndustryClassification)
		WITH o, { id:ind.uuid, types:labels(ind), prefLabel:ind.prefLabel} as ind
		WITH o, ind
		OPTIONAL MATCH (lei:LegalEntityIdentifier)-[:IDENTIFIES]->(o)
		WITH o, ind, { legalEntityIdentifier:lei.value } as lei
		WITH o, ind, lei
		OPTIONAL MATCH (o)-[:SUB_ORGANISATION_OF]->(parent:Organisation)
		WITH o, ind, lei, { id:parent.uuid, types:labels(parent), prefLabel:parent.prefLabel} as parent
		WITH o, ind, lei, parent
		OPTIONAL MATCH (o)<-[:ISSUED_BY]-(fi:FinancialInstrument)<-[:IDENTIFIES]-(figi:FIGIIdentifier)
		WITH o, ind, lei, parent, {id:fi.uuid, types:labels(fi), prefLabel:fi.prefLabel, figi:figi.value} as fi
		WITH o, ind, lei, parent, fi
		WITH o, ind, lei, parent, fi
		OPTIONAL MATCH (o)<-[:SUB_ORGANISATION_OF]-(sub:Organisation)
		WITH o, ind, lei, parent, fi, sub, size((:Content)-[:MENTIONS]->(sub)) as annCounts ORDER BY annCounts DESC, o.prefLabel ASC
		WITH o, ind, lei, parent, fi, { id:sub.uuid, types:labels(sub), prefLabel:sub.prefLabel, annCount:annCounts } as sub
		WITH o, ind, lei, parent, fi, collect(sub) as sub
		return {id:o.uuid, types:labels(o), prefLabel:o.prefLabel, labels:o.aliases, lei:lei, parent:parent, ind:ind, sub:sub, fi:fi} as rs`,
		Parameters: neoism.Props{"uuid": uuid},
		Result:     &results,
	}

	if err := pcw.conn.CypherBatch([]*neoism.CypherQuery{query}); err != nil || len(results) == 0 {
		return Organisation{}, false, err
	} else if len(results) != 1 {
		errMsg := fmt.Sprintf("Multiple organisations found with the same uuid:%s !", uuid)
		log.Error(errMsg)
		return Organisation{}, true, errors.New(errMsg)
	}

	organisation = neoReadStructToOrganisation(results[0].Rs, pcw.env)
	return organisation, true, nil
}

func neoReadStructToOrganisation(neo neoReadStruct, env string) Organisation {
	public := Organisation{}
	public.Thing = &Thing{}
	public.ID = mapper.IDURL(neo.ID)
	public.APIURL = mapper.APIURL(neo.ID, neo.Types, env)
	public.Types = mapper.TypeURIs(neo.Types)
	public.DirectType = filterToMostSpecificType(neo.Types)
	public.PrefLabel = neo.PrefLabel
	public.ProperName = neo.ProperName
	public.ShortName = neo.ShortName
	public.HiddenLabel = neo.HiddenLabel
	public.FormerNames = neo.FormerNames
	public.CountryCode = neo.CountryCode
	public.CountryOfIncorporation = neo.CountryOfIncorporation
	public.PostalCode = neo.PostalCode
	public.YearFounded = neo.YearFounded
	if len(neo.Labels) > 0 {
		public.Labels = &neo.Labels
	}

	if neo.Lei.LegalEntityIdentifier != "" {
		public.LegalEntityIdentifier = neo.Lei.LegalEntityIdentifier
	}

	if neo.Ind.ID != "" {
		public.IndustryClassification = &IndustryClassification{}
		public.IndustryClassification.Thing = &Thing{}
		public.IndustryClassification.ID = mapper.IDURL(neo.Ind.ID)
		public.IndustryClassification.APIURL = mapper.APIURL(neo.Ind.ID, neo.Ind.Types, env)
		public.IndustryClassification.Types = mapper.TypeURIs(neo.Ind.Types)
		public.IndustryClassification.DirectType = filterToMostSpecificType(neo.Ind.Types)
		public.IndustryClassification.PrefLabel = neo.Ind.PrefLabel
	}

	if neo.Fi.ID != "" {
		public.FinancialInstrument = &FinancialInstrument{}
		public.FinancialInstrument.Thing = &Thing{}
		public.FinancialInstrument.ID = mapper.IDURL(neo.Fi.ID)
		public.FinancialInstrument.APIURL = mapper.APIURL(neo.Fi.ID, neo.Fi.Types, env)
		public.FinancialInstrument.Types = mapper.TypeURIs(neo.Fi.Types)
		public.FinancialInstrument.DirectType = filterToMostSpecificType(neo.Fi.Types)
		public.FinancialInstrument.PrefLabel = neo.Fi.PrefLabel
		public.FinancialInstrument.Figi = neo.Fi.FIGI
	}

	if neo.Parent.ID != "" {
		public.Parent = &Parent{}
		public.Parent.Thing = &Thing{}
		public.Parent.ID = mapper.IDURL(neo.Parent.ID)
		public.Parent.APIURL = mapper.APIURL(neo.Parent.ID, neo.Parent.Types, env)
		public.Parent.Types = mapper.TypeURIs(neo.Parent.Types)
		public.Parent.DirectType = filterToMostSpecificType(neo.Parent.Types)
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
			subsidiary.DirectType = filterToMostSpecificType(neoSub.Types)
			subsidiary.PrefLabel = neoSub.PrefLabel
			public.Subsidiaries[idx] = subsidiary
		}
	}

	log.Debugf("neoReadStructToOrganisation neo: %+v result: %+v", neo, public)
	return public
}

func filterToMostSpecificType(unfilteredTypes []string) string {
	mostSpecificType, err := mapper.MostSpecificType(unfilteredTypes)
	if err != nil {
		return ""
	}
	fullURI := mapper.TypeURIs([]string{mostSpecificType})
	return fullURI[0]
}
