package organisations

// Thing is the base entity, all nodes in neo4j should have these properties
/* The following is currently defined in Java (3da1b900b38)
@JsonInclude(NON_EMPTY)
public class Thing {
    public String id;
    public String apiUrl;
    public String prefLabel;
    public List<String> types = new ArrayList<>();
}
*/
type Thing struct {
	ID        string `json:"id"`
	APIURL    string `json:"apiUrl"` // self ?
	PrefLabel string `json:"prefLabel,omitempty"`
}

// Organisation is the structure used for the people API
/* The following is currently defined in Java (e4b93668e32) but I think we should be removing profile
@JsonInclude(NON_EMPTY)
public class Organisation extends Thing {

    public List<String> labels = new ArrayList<>();
    public String profile;
    public Thing industryClassification;
    public Thing parentOrganisation;
    public List<Thing> subsidiaries = new ArrayList<>();
    public List<Membership> memberships = new ArrayList<>(); - except membership, which has been removed from the response
}
*/
type Organisation struct {
	*Thing
	ProperName             string                  `json:"properName,omitempty"`
	ShortName              string                  `json:"shortName,omitempty"`
	HiddenLabel            string                  `json:"hiddenLabel,omitempty"`
	FormerNames            []string                `json:"formerNames,omitempty"`
	CountryCode            string                  `json:"countryCode,omitempty"`
	CountryOfIncorporation string                  `json:"countryOfIncorporation,omitempty"`
	PostalCode             string                  `json:"postalCode,omitempty"`
	YearFounded            int                     `json:"yearFounded,omitempty"`
	Types                  []string                `json:"types"`
	DirectType             string                  `json:"directType,omitempty"`
	Labels                 *[]string               `json:"labels,omitempty"`
	LegalEntityIdentifier  string                  `json:"leiCode,omitempty"`
	IndustryClassification *IndustryClassification `json:"industryClassification,omitempty"` //this is a pointer so that the struct is omitted if empty
	Parent                 *Parent                 `json:"parentOrganisation,omitempty"`
	Subsidiaries           []Subsidiary            `json:"subsidiaries,omitempty"`
	FinancialInstrument    *FinancialInstrument    `json:"financialInstrument,omitempty"`
}

// Parent is a simplified representation of a parent organisation, used in Organisation API
type Parent struct {
	*Thing
	Types      []string `json:"types,omitempty"`
	DirectType string   `json:"directType,omitempty"`
}

// Subsidiary is a simplified representation of a subsidiary organisation, used in Organisation API
type Subsidiary struct {
	*Thing
	Types      []string `json:"types,omitempty"`
	DirectType string   `json:"directType,omitempty"`
}

// IndustryClassification represents the type of Organisation, e.g. a Bank
type IndustryClassification struct {
	*Thing
	Types      []string `json:"types,omitempty"`
	DirectType string   `json:"directType,omitempty"`
}

type FinancialInstrument struct {
	*Thing
	Types      []string `json:"types,omitempty"`
	DirectType string   `json:"directType,omitempty"`
	Figi       string   `json:"FIGI"`
}
