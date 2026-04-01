// Package hal provides HAL+JSON helpers for HATEOAS responses.
// See https://datatracker.ietf.org/doc/html/draft-kelly-json-hal
package hal

import (
	"encoding/json"
	"net/http"
)

// ContentType is the MIME type for HAL+JSON.
const ContentType = "application/hal+json"

// Link represents a HAL link object.
type Link struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated,omitempty"`
	Title     string `json:"title,omitempty"`
	Type      string `json:"type,omitempty"`
}

// Links is a map of link relations to Link objects.
type Links map[string]interface{}

// NewLink creates a simple link with just an href.
func NewLink(href string) Link {
	return Link{Href: href}
}

// NewTemplatedLink creates a templated link.
func NewTemplatedLink(href string) Link {
	return Link{Href: href, Templated: true}
}

// NewTitledLink creates a link with a title.
func NewTitledLink(href, title string) Link {
	return Link{Href: href, Title: title}
}

// Resource wraps any data with HAL _links.
// Use for adding _links to existing response structures.
type Resource struct {
	Links    Links       `json:"_links,omitempty"`
	Embedded interface{} `json:"_embedded,omitempty"`
	Data     interface{} `json:"-"` // Will be flattened
}

// MarshalJSON flattens Data fields into the resource object.
func (r Resource) MarshalJSON() ([]byte, error) {
	// First, marshal the data to get its fields
	dataBytes, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}

	// Parse data as a map
	var dataMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		// Data might not be a map, just return links + data as-is
		return json.Marshal(struct {
			Links    Links       `json:"_links,omitempty"`
			Embedded interface{} `json:"_embedded,omitempty"`
			Data     interface{} `json:"data,omitempty"`
		}{
			Links:    r.Links,
			Embedded: r.Embedded,
			Data:     r.Data,
		})
	}

	// Add _links to the map
	if r.Links != nil {
		dataMap["_links"] = r.Links
	}

	// Add _embedded if present
	if r.Embedded != nil {
		dataMap["_embedded"] = r.Embedded
	}

	return json.Marshal(dataMap)
}

// Builder helps construct HAL responses with method chaining.
type Builder struct {
	links    Links
	embedded interface{}
	data     interface{}
}

// NewBuilder creates a new HAL resource builder.
func NewBuilder() *Builder {
	return &Builder{
		links: make(Links),
	}
}

// Self adds a self link.
func (b *Builder) Self(href string) *Builder {
	b.links["self"] = NewLink(href)
	return b
}

// Link adds a named link.
func (b *Builder) Link(rel, href string) *Builder {
	b.links[rel] = NewLink(href)
	return b
}

// LinkWithTitle adds a named link with a title.
func (b *Builder) LinkWithTitle(rel, href, title string) *Builder {
	b.links[rel] = NewTitledLink(href, title)
	return b
}

// TemplatedLink adds a templated link.
func (b *Builder) TemplatedLink(rel, href string) *Builder {
	b.links[rel] = NewTemplatedLink(href)
	return b
}

// Links adds multiple links at once.
func (b *Builder) Links(links Links) *Builder {
	for k, v := range links {
		b.links[k] = v
	}
	return b
}

// Embedded adds embedded resources.
func (b *Builder) Embedded(embedded interface{}) *Builder {
	b.embedded = embedded
	return b
}

// Data sets the main data payload.
func (b *Builder) Data(data interface{}) *Builder {
	b.data = data
	return b
}

// Build returns the constructed Resource.
func (b *Builder) Build() Resource {
	return Resource{
		Links:    b.links,
		Embedded: b.embedded,
		Data:     b.data,
	}
}

// WriteJSON writes a HAL+JSON response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteResource writes a HAL resource as JSON.
func WriteResource(w http.ResponseWriter, status int, resource Resource) {
	WriteJSON(w, status, resource)
}

// AddLinksToMap adds _links to an existing map.
// Useful for augmenting existing response structures without changing types.
func AddLinksToMap(m map[string]interface{}, links Links) map[string]interface{} {
	if m == nil {
		m = make(map[string]interface{})
	}
	m["_links"] = links
	return m
}
