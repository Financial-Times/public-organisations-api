package organisation

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
    public List<Membership> memberships = new ArrayList<>();
}
*/
type Organisation struct {
	*Thing
	Types                  []string                `json:"types"`
	LEICode                string                  `json:"leiCode,omitempty"`
	Labels                 *[]string               `json:"labels,omitempty"`
	IndustryClassification *IndustryClassification `json:"industryClassification,omitempty"` //this is a pointer so that the struct is omitted if empty
	Parent                 *Parent                 `json:"parentOrganisation,omitempty"`
	Subsidiaries           []Subsidiary            `json:"subsidiaries,omitempty"`
	Memberships            []Membership            `json:"memberships,omitempty"`
}

// Membership represents the relationship between an organisation and a person
/*
@JsonInclude(Include.NON_EMPTY)
public class Membership {
    public String title;
    public Thing organisation;
    public Thing person;
    public List<ChangeEvent> changeEvents = new ArrayList();
    public List<MembershipRole> roles = new ArrayList();
*/
type Membership struct {
	Title        string         `json:"title,omitempty"`
	Person       Person         `json:"person"`
	ChangeEvents *[]ChangeEvent `json:"changeEvents,omitempty"`
}

// Person simplified representation used in Organisation API
type Person struct {
	*Thing
	Types []string `json:"types,omitempty"`
}

// Parent is a simplified representation of a parent organisation, used in Organisation API
type Parent struct {
	*Thing
	Types []string `json:"types,omitempty"`
}

// Subsidiary is a simplified representation of a subsidiary organisation, used in Organisation API
type Subsidiary struct {
	*Thing
	Types []string `json:"types,omitempty"`
}

// IndustryClassification represents the type of Organisation, e.g. a Bank
type IndustryClassification struct {
	*Thing
	Types []string `json:"types,omitempty"`
}

// ChangeEvent represent when something started or ended
/*
@JsonInclude(Include.NON_EMPTY)
public class ChangeEvent {
    public String startedAt;
    public String endedAt;
*/
type ChangeEvent struct {
	StartedAt string `json:"startedAt,omitempty"`
	EndedAt   string `json:"endedAt,omitempty"`
}
