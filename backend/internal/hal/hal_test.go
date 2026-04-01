package hal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLink(t *testing.T) {
	link := NewLink("/api/test")
	if link.Href != "/api/test" {
		t.Errorf("expected href /api/test, got %s", link.Href)
	}
	if link.Templated {
		t.Error("expected Templated to be false")
	}
}

func TestNewTemplatedLink(t *testing.T) {
	link := NewTemplatedLink("/api/users/{id}")
	if link.Href != "/api/users/{id}" {
		t.Errorf("expected href /api/users/{id}, got %s", link.Href)
	}
	if !link.Templated {
		t.Error("expected Templated to be true")
	}
}

func TestNewTitledLink(t *testing.T) {
	link := NewTitledLink("/api/test", "Test Resource")
	if link.Href != "/api/test" {
		t.Errorf("expected href /api/test, got %s", link.Href)
	}
	if link.Title != "Test Resource" {
		t.Errorf("expected title 'Test Resource', got %s", link.Title)
	}
}

func TestBuilder(t *testing.T) {
	resource := NewBuilder().
		Self("/api/components/layout").
		Link("collection", "/api/components").
		LinkWithTitle("ds:versions", "/api/components/layout/versions", "Versions").
		TemplatedLink("ds:version", "/api/components/layout/{version}").
		Data(map[string]string{"name": "layout"}).
		Build()

	// Verify links
	if len(resource.Links) != 4 {
		t.Errorf("expected 4 links, got %d", len(resource.Links))
	}

	selfLink, ok := resource.Links["self"].(Link)
	if !ok {
		t.Fatal("self link not found or wrong type")
	}
	if selfLink.Href != "/api/components/layout" {
		t.Errorf("expected self href /api/components/layout, got %s", selfLink.Href)
	}

	versionLink, ok := resource.Links["ds:version"].(Link)
	if !ok {
		t.Fatal("ds:version link not found or wrong type")
	}
	if !versionLink.Templated {
		t.Error("expected ds:version link to be templated")
	}
}

func TestResourceMarshalJSON(t *testing.T) {
	resource := Resource{
		Links: Links{
			"self":       NewLink("/api/test"),
			"collection": NewLink("/api/tests"),
		},
		Data: map[string]interface{}{
			"id":   "123",
			"name": "test",
		},
	}

	bytes, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify _links is present
	links, ok := result["_links"].(map[string]interface{})
	if !ok {
		t.Fatal("_links not found in result")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}

	// Verify data fields are flattened
	if result["id"] != "123" {
		t.Errorf("expected id '123', got %v", result["id"])
	}
	if result["name"] != "test" {
		t.Errorf("expected name 'test', got %v", result["name"])
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"_links": Links{
			"self": NewLink("/api/test"),
		},
		"name": "test",
	}

	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != ContentType {
		t.Errorf("expected content type %s, got %s", ContentType, contentType)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("expected name 'test', got %v", result["name"])
	}
}

func TestAddLinksToMap(t *testing.T) {
	m := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}

	links := Links{
		"self": NewLink("/api/test/123"),
	}

	result := AddLinksToMap(m, links)

	if result["id"] != "123" {
		t.Errorf("expected id '123', got %v", result["id"])
	}

	resultLinks, ok := result["_links"].(Links)
	if !ok {
		t.Fatal("_links not found or wrong type")
	}

	selfLink, ok := resultLinks["self"].(Link)
	if !ok {
		t.Fatal("self link not found or wrong type")
	}
	if selfLink.Href != "/api/test/123" {
		t.Errorf("expected self href /api/test/123, got %s", selfLink.Href)
	}
}

func TestAddLinksToMap_NilMap(t *testing.T) {
	links := Links{
		"self": NewLink("/api/test"),
	}

	result := AddLinksToMap(nil, links)

	if result == nil {
		t.Fatal("expected non-nil map")
	}

	_, ok := result["_links"]
	if !ok {
		t.Error("_links not found in result")
	}
}
