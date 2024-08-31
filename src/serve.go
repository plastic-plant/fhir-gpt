// serve.go
//
// Serves content from /database folder as a simple FHIR server.
// Adds a generated summary with custom resource as a proof-of-concept.
// Bundles if query returns multiple resources, and simple faux pas use
// of the _summary=true switch. ;-)
//
// go run serve.go R:\fhir-gpt\database
// GOOS=linux GOARCH=amd64 go build -o ../util/serve serve.go

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var databaseFolder = "../database"

type BundleJsonResponse struct {
	ResourceType string        `json:"resourceType"`
	Entry        []BundleEntry `json:"entry"`
	Summary      string        `json:"summary,omitempty"`
}

type BundleEntry struct {
	Resource ResourceJsonResponse `json:"resource"`
}

type ResourceJsonResponse map[string]interface{}

type ErrorJsonResponse struct {
	Error string `json:"error"`
}

func main() {
	if len(os.Args) > 0 {
		databaseFolder = os.Args[1]
	} else {
		fmt.Println("Usage:   serve <database folder>")
		fmt.Println("Example: serve")
		fmt.Println("Example: serve ../database")
		fmt.Println("Example: go run serve.go ~/fhir-gpt/database")
		fmt.Println("Example: go run serve.go C:\\Program Files\\fhir-gpt\\database")
		fmt.Println("Try:     http://localhost/fhir/Patient")
		fmt.Println("Try:     http://localhost/fhir/Patient/nl-core-Patient-01?_summary=true&_include=AllergyIntolerance,EpisodeOfCare")
		fmt.Println()
		return
	}

	log.Println("Starting FHIR server on localhost:80. Database:", databaseFolder)
	http.HandleFunc("/fhir/", fhirHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func fhirHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Take resourceType and optional resourceId from the url http://localhost/fhir/ResourceType/ResourceId, eg
	// http://localhost/fhir/Patient/nl-core-Patient-01?_summary=true&_include=AllergyIntolerance,EpisodeOfCare
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/fhir/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid path, expecting /fhir/ResourceType/{ResourceId}", http.StatusBadRequest)
		return
	}
	resourceType := pathParts[0]
	var resourceId string
	if len(pathParts) > 1 {
		resourceId = pathParts[1]
	}

	// Load content for requested resource type, matching files in database into a dictionary.
	resourceMap := make(map[string]ResourceJsonResponse)
	loadResources(resourceMap, resourceType, resourceId)

	// Load additional resource types in bundled response when optional _include is given in url.
	if len(resourceMap) > 0 {
		includeOptionalLinkedResourceTypes := strings.Split(r.URL.Query().Get("_include"), ",")
		for _, includeResourceType := range includeOptionalLinkedResourceTypes {
			if includeResourceType != "" {
				loadResources(resourceMap, includeResourceType, "")
			}
		}
	}

	// Add generated custom summary resource in bundle, unless _summary=false is given in url.
	summary := r.URL.Query().Get("_summary") != "false"
	if summary && len(resourceMap) > 0 {
		includeSummary(resourceMap)
	}

	// Return operation outcome, a single resource or bundle response.
	switch len(resourceMap) {
	case 0:
		writeErrorJson(w, "not-found")
	case 1:
		writeResourceJson(w, resourceMap)
	default:
		writeBundleJson(w, resourceMap)
	}

	log.Println(fmt.Sprintf("[%s] Request for %s returned %d resources.", time.Since(startTime), r.URL.Path, len(resourceMap)))
}

func loadResources(resourceMap map[string]ResourceJsonResponse, resourceType, resourceId string) {
	resourceFolder := filepath.Join(databaseFolder, resourceType)

	// Skip loading if resource folder does not exist.
	if _, err := os.Stat(resourceFolder); os.IsNotExist(err) {
		return
	}

	// Load resource content if resourceId is specified (e.g. Patient/patient-123),
	// else return all resources from folder matching the resource type (e.g. Patient).
	if resourceId != "" {
		resourceFile := filepath.Join(resourceFolder, resourceId+".json")
		if content := getResourceFromFile(resourceFile); content != nil {
			resourceMap[resourceId] = content
		}
	} else {
		files, _ := ioutil.ReadDir(resourceFolder)
		for _, file := range files {
			if !file.IsDir() {
				resourceFile := filepath.Join(resourceFolder, file.Name())
				if content := getResourceFromFile(resourceFile); content != nil {
					resourceMap[file.Name()] = content
				}
			}
		}
	}
}

func getResourceFromFile(filePath string) ResourceJsonResponse {
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil
	}

	return convertToResource(string(fileContent))
}

func convertToResource(content string) ResourceJsonResponse {
	var resourceContent ResourceJsonResponse
	if err := json.Unmarshal([]byte(content), &resourceContent); err != nil {
		fmt.Println("Error unmarshalling JSON from file")
		return nil
	}
	return resourceContent
}

func includeSummary(resourceMap map[string]ResourceJsonResponse) {
	summary := make(ResourceJsonResponse)
	summary["resourceType"] = "Summary"
	summary["subject"] = "Patient/nl-core-Patient-01"
	summary["date"] = "2024-01-01"
	summary["author"] = "GPT"
	summary["content"] = "This is a summary of the patient record."
	resourceMap["summary"] = summary
}

func writeResourceJson(w http.ResponseWriter, resourceMap map[string]ResourceJsonResponse) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(http.StatusOK)

	for _, resource := range resourceMap {
		json, err := json.MarshalIndent(resource, "", "  ")
		if err != nil {
			writeErrorJson(w, "exception")
			return
		}

		w.Write(json)
		break
	}
}

func writeBundleJson(w http.ResponseWriter, resourceMap map[string]ResourceJsonResponse) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(http.StatusOK)

	var bundle BundleJsonResponse
	bundle.ResourceType = "Bundle"
	for _, resource := range resourceMap {
		bundle.Entry = append(bundle.Entry, BundleEntry{Resource: resource})
	}

	json, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		writeErrorJson(w, "exception")
		return
	}

	w.Write(json)
}

func writeErrorJson(w http.ResponseWriter, issueTypeCode string) {
	w.Header().Set("Content-Type", "application/fhir+json")
	w.WriteHeader(http.StatusOK)

	errorJson := strings.Replace(`{
		"resource": {
			"resourceType": "OperationOutcome",
			"issue": [
				{
					"severity": "error",
					"code": "{issueTypeCode}"
				}
			]
		}
	}`, "{issueTypeCode}", issueTypeCode, 1)

	w.Write([]byte(errorJson))
}
