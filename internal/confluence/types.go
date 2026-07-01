package confluence

// Page is the subset of the Confluence Cloud v2 page object atlit renders.
type Page struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Title     string   `json:"title"`
	SpaceID   string   `json:"spaceId"`
	AuthorID  string   `json:"authorId"`
	CreatedAt string   `json:"createdAt"`
	Version   *Version `json:"version"`
	Body      Body     `json:"body"`
	Links     Links    `json:"_links"`
}

// Version holds the page version metadata.
type Version struct {
	Number    int    `json:"number"`
	CreatedAt string `json:"createdAt"`
	AuthorID  string `json:"authorId"`
	Message   string `json:"message"`
}

// Body wraps the requested body representation(s).
type Body struct {
	AtlasDocFormat Representation `json:"atlas_doc_format"`
}

// Representation is one body rendering. For atlas_doc_format, Value is an ADF
// document encoded as a JSON string.
type Representation struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

// Links holds the page's relative web links.
type Links struct {
	WebUI string `json:"webui"`
	Base  string `json:"base"`
}

// Attachment is a file attached to a Confluence page. DownloadLink is relative
// to the site (e.g. /download/attachments/...); the renderer resolves it to an
// absolute URL against the page's link base.
type Attachment struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	MediaType    string `json:"mediaType"`
	FileSize     int    `json:"fileSize"`
	DownloadLink string `json:"downloadLink"`
}

// attachmentList is one page of the v2 attachments listing.
type attachmentList struct {
	Results []Attachment `json:"results"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}
