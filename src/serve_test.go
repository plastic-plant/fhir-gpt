package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDatabase(t *testing.T) string {
	t.Helper()

	tempDir, err := ioutil.TempDir("", "test-database")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	resourceTypes := map[string]string{
		"Patient":            `{"resourceType": "Patient", "id": "patient-123", "name": [{"given": ["John"], "family": "Doe"}]}`,
		"AllergyIntolerance": `{"resourceType": "AllergyIntolerance", "id": "allergy-456", "substance": {"text": "Peanuts"}}`,
	}

	for resourceType, content := range resourceTypes {
		resourceDir := filepath.Join(tempDir, resourceType)
		if err := os.Mkdir(resourceDir, 0755); err != nil {
			t.Fatalf("Failed to create resource directory: %v", err)
		}

		resourceFile := filepath.Join(resourceDir, "patient-123.json")
		if err := ioutil.WriteFile(resourceFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test resource file: %v", err)
		}
	}

	return tempDir
}

func cleanupTestDatabase(tempDir string) {
	os.RemoveAll(tempDir)
}

func TestFhirHandlerSingleResource(t *testing.T) {
	tempDir := setupTestDatabase(t)
	defer cleanupTestDatabase(tempDir)

	databaseFolder = tempDir

	req, err := http.NewRequest("GET", "/fhir/Patient/patient-123", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(fhirHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resource ResourceJsonResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resource)
	if err != nil {
		t.Errorf("Error unmarshalling response body: %v", err)
	}

	if resource["resourceType"] != "Patient" {
		t.Errorf("handler returned wrong resourceType: got %v want %v", resource["resourceType"], "Patient")
	}
}

func TestFhirHandlerBundleResource(t *testing.T) {
	tempDir := setupTestDatabase(t)
	defer cleanupTestDatabase(tempDir)
	databaseFolder = tempDir

	req, err := http.NewRequest("GET", "/fhir/Patient?_include=AllergyIntolerance", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(fhirHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var bundle BundleJsonResponse
	err = json.Unmarshal(rr.Body.Bytes(), &bundle)
	if err != nil {
		t.Errorf("Error unmarshalling response body: %v", err)
	}

	if bundle.ResourceType != "Bundle" {
		t.Errorf("handler returned wrong resourceType: got %v want %v", bundle.ResourceType, "Bundle")
	}

	if len(bundle.Entry) != 2 {
		t.Errorf("handler returned wrong number of entries: got %v want %v", len(bundle.Entry), 2)
	}
}

func TestFhirHandlerInvalidPath(t *testing.T) {
	req, err := http.NewRequest("GET", "/fhir/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(fhirHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestFhirHandlerNotFound(t *testing.T) {
	tempDir := setupTestDatabase(t)
	defer cleanupTestDatabase(tempDir)

	databaseFolder = tempDir
	req, err := http.NewRequest("GET", "/fhir/UnknownResource", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(fhirHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), `"code": "not-found"`) {
		t.Errorf("handler returned wrong error response: %v", rr.Body.String())
	}
}

func TestFhirHandlerSummary(t *testing.T) {
	tempDir := setupTestDatabase(t)
	defer cleanupTestDatabase(tempDir)
	databaseFolder = tempDir

	req, err := http.NewRequest("GET", "/fhir/Patient/patient-123?_summary=true", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(fhirHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resource ResourceJsonResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resource)
	if err != nil {
		t.Errorf("Error unmarshalling response body: %v", err)
	}

	if _, ok := resource["summary"]; !ok {
		t.Errorf("handler did not include summary in response")
	}
}
