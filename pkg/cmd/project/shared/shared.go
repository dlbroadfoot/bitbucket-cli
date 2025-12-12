package shared

// Project represents a Bitbucket project
type Project struct {
	Key         string `json:"key"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Owner       *Owner `json:"owner,omitempty"`
	Workspace   *Owner `json:"workspace,omitempty"`
	Links       Links  `json:"links"`
}

type Owner struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	Slug        string `json:"slug"`
	Links       Links  `json:"links"`
}

type Links struct {
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	Avatar struct {
		Href string `json:"href"`
	} `json:"avatar"`
}

// ProjectList represents a paginated list of projects
type ProjectList struct {
	Size     int       `json:"size"`
	Page     int       `json:"page"`
	PageLen  int       `json:"pagelen"`
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Values   []Project `json:"values"`
}

// HTMLURL returns the web URL for the project
func (p *Project) HTMLURL() string {
	return p.Links.HTML.Href
}
