package discovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersion(t *testing.T) {
	// Clear any existing env var
	t.Setenv("APP_VERSION", "")

	// Test default version
	v := Version()
	if v != "1.0.0" {
		t.Errorf("expected default version 1.0.0, got %s", v)
	}

	// Test with env var
	t.Setenv("APP_VERSION", "2.0.0")
	v = Version()
	if v != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", v)
	}
}

func TestServiceName(t *testing.T) {
	// Clear any existing env var
	t.Setenv("SERVICE_NAME", "")

	// Test default service name
	s := ServiceName()
	if s != "ds-app-template" {
		t.Errorf("expected default service name ds-app-template, got %s", s)
	}

	// Test with env var
	t.Setenv("SERVICE_NAME", "custom-service")
	s = ServiceName()
	if s != "custom-service" {
		t.Errorf("expected service name custom-service, got %s", s)
	}
}

func TestBuildDiscoveryResponse(t *testing.T) {
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("APP_VERSION", "")

	resp := BuildDiscoveryResponse()

	// Check service and version
	if resp.Service != "ds-app-template" {
		t.Errorf("expected service ds-app-template, got %s", resp.Service)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", resp.Version)
	}

	// Check _links
	if resp.Links == nil {
		t.Fatal("expected _links to be set")
	}

	// Check self link
	self, ok := resp.Links["self"].(HALLink)
	if !ok {
		t.Fatal("expected self link to be HALLink")
	}
	if self.Href != "/api/discovery" {
		t.Errorf("expected self href /api/discovery, got %s", self.Href)
	}

	// Check curies
	curies, ok := resp.Links["curies"].([]HALCurie)
	if !ok {
		t.Fatal("expected curies to be []HALCurie")
	}
	if len(curies) != 1 {
		t.Errorf("expected 1 curie, got %d", len(curies))
	}
	if curies[0].Name != "ds" {
		t.Errorf("expected curie name ds, got %s", curies[0].Name)
	}
	if !curies[0].Templated {
		t.Error("expected curie to be templated")
	}

	// Check required endpoints exist
	requiredEndpoints := []string{
		"ds:health",
		"ds:session",
		"ds:theme",
		"ds:auth-login",
		"ds:auth-logout",
		"ds:me",
		"ds:tenant",
		"ds:flags-evaluate",
		"ds:flags",
		"ds:components",
	}

	for _, endpoint := range requiredEndpoints {
		if _, ok := resp.Links[endpoint]; !ok {
			t.Errorf("expected endpoint %s to be present in _links", endpoint)
		}
	}
}

func TestHandler_Success(t *testing.T) {
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("APP_VERSION", "")

	req := httptest.NewRequest(http.MethodGet, "/api/discovery", nil)
	w := httptest.NewRecorder()

	handler := Handler(nil)
	handler(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != ContentTypeHALJSON {
		t.Errorf("expected Content-Type %s, got %s", ContentTypeHALJSON, contentType)
	}

	// Check response body is valid JSON
	var resp DiscoveryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	if resp.Service != "ds-app-template" {
		t.Errorf("expected service ds-app-template, got %s", resp.Service)
	}
}

func TestHandler_WithAppLinks(t *testing.T) {
	appLinks := func() map[string]interface{} {
		return map[string]interface{}{
			"ds:custom": HALLink{Href: "/api/custom"},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discovery", nil)
	w := httptest.NewRecorder()

	handler := Handler(appLinks)
	handler(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}

	links := resp["_links"].(map[string]interface{})
	if _, ok := links["ds:custom"]; !ok {
		t.Error("expected app-specific link ds:custom to be present")
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/discovery", nil)
			w := httptest.NewRecorder()

			handler := Handler(nil)
			handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, w.Code)
			}

			// Check Allow header
			allow := w.Header().Get("Allow")
			if allow != "GET" {
				t.Errorf("expected Allow header GET, got %s", allow)
			}
		})
	}
}

func TestHandler_HALStructure(t *testing.T) {
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("APP_VERSION", "")

	req := httptest.NewRequest(http.MethodGet, "/api/discovery", nil)
	w := httptest.NewRecorder()

	handler := Handler(nil)
	handler(w, req)

	// Parse response as generic map to verify HAL structure
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// HAL requires _links
	links, ok := resp["_links"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _links to be present and be an object")
	}

	// Check self link exists
	if _, ok := links["self"]; !ok {
		t.Error("HAL requires self link")
	}

	// Check curies exist
	curies, ok := links["curies"].([]interface{})
	if !ok {
		t.Fatal("expected curies to be present")
	}
	if len(curies) == 0 {
		t.Error("expected at least one curie")
	}

	// Verify curie structure
	curie, ok := curies[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected curie to be an object")
	}
	if curie["name"] != "ds" {
		t.Errorf("expected curie name ds, got %v", curie["name"])
	}
	if curie["templated"] != true {
		t.Errorf("expected curie to be templated")
	}
	expectedHref := "https://developer.digistratum.com/docs/rels/{rel}"
	if curie["href"] != expectedHref {
		t.Errorf("expected curie href %s, got %v", expectedHref, curie["href"])
	}

	// Check service and version at root (not nested)
	if resp["service"] != "ds-app-template" {
		t.Errorf("expected service at root level")
	}
	if resp["version"] != "1.0.0" {
		t.Errorf("expected version at root level")
	}
}

func TestWriteHALJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	WriteHALJSON(w, http.StatusOK, data)

	// Check Content-Type
	if w.Header().Get("Content-Type") != ContentTypeHALJSON {
		t.Errorf("expected Content-Type %s", ContentTypeHALJSON)
	}

	// Check status
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
