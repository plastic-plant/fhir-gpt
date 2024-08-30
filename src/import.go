// import.go
//
// Reads /examples folder and writes all XML and JSON files to /database folder.
// Expects a FHIR resource in XML or JSON with resource name and resource id.
// Converts to JSON and names the file as database resourcetype/resourceid.json.
//
// GOOS=linux GOARCH=amd64 go build -o ../util/import import.go

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type XMLRoot struct {
	XMLName xml.Name `xml:""`
	ID struct {
		Value string `xml:"value,attr"`
	} `xml:"id"`
}

type JSONRoot map[string]json.RawMessage

type JSONID struct {
	Value string `json:"_value"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:   import <examples folder> <database folder>")
		fmt.Println("Example: import ../examples ../database")
		fmt.Println("Example: go run import.go ../examples ../database")
		fmt.Println()
		os.Exit(0)
	}

	examplesFolder := os.Args[1]
	databaseFolder := os.Args[2]

	filepath.Walk(examplesFolder, func(path string, info fs.FileInfo, err error) {
		if err != nil {
			fmt.Printf("Error reading file %v", err)
			os.Exit(1)
		}

		if !info.IsDir() {
			resourceName, resourceId, resourceData := processFile(path)
			if resourceName == "" || resourceId == "" || resourceData == nil {
				return
			}

			databaseResourceFolder := filepath.Join(databaseFolder, resourceName)
			if err := os.MkdirAll(databaseResourceFolder, os.ModePerm); err != nil {
				fmt.Printf("Error creating folder %s: %v", databaseResourceFolder, err)
				os.Exit(1)
			}

			databaseResourcePath := filepath.Join(databaseResourceFolder, resourceId + ".json")
			saveAsJSON(databaseResourcePath, resourceData)
		}
	})
}

func processFile(path string) (string, string, map[string]interface{}) {
	if strings.HasSuffix(path, ".xml") {
		return processXML(path)
	} else if strings.HasSuffix(path, ".json") {
		return processJSON(path)
	}
	return "", "", nil
}

func processXML(path string) (string, string, map[string]interface{}) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading file %s: %v", path, err)
		return "", "", nil
	}

	var root XMLRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		fmt.Printf("Error parsing file; is this XML? %s: %v", path, err)
		return "", "", nil
	}

	rootName := root.XMLName.Local
	id := root.ID.Value

	if rootName == "" || id == "" {
		fmt.Printf("Error reading root element name or logical id from resource; is this XML? %s", path)
		return "", "", nil
	}

	generic, err := xmlToMap(data)
	if err != nil {
		fmt.Printf("Error reading XML as map for %s: %v", path, err)
		return "", "", nil
	}

	return rootName, id, map[string]interface{}{rootName: generic}
}

func xmlToMap(data []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := xml.Unmarshal(data, (*mapToXMLStruct)(&m)); err != nil {
		return nil, err
	}
	return m, nil
}

type mapToXMLStruct map[string]interface{}

func (m *mapToXMLStruct) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*m = map[string]interface{}{start.Name.Local: map[string]interface{}{}}
	currentMap := *m

	var stack []map[string]interface{}
	stack = append(stack, currentMap[start.Name.Local].(map[string]interface{}))

	for {
		t, err := d.Token()
		if err != nil {
			return err
		}

		switch elem := t.(type) {
		case xml.StartElement:
			node := map[string]interface{}{}
			for _, attr := range elem.Attr {
				node["@"+attr.Name.Local] = attr.Value
			}
			stack[len(stack)-1][elem.Name.Local] = node
			stack = append(stack, node)

		case xml.EndElement:
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return nil
			}

		case xml.CharData:
			if s := strings.TrimSpace(string(elem)); len(s) > 0 {
				stack[len(stack)-1]["#text"] = s
			}
		}
	}
}

func processJSON(path string) (string, string, map[string]interface{}) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading file %s: %v", path, err)
		return "", "", nil
	}

	var root JSONRoot
	if err := json.Unmarshal(data, &root); err != nil {
		fmt.Printf("Error parsing file; is this JSON? %s: %v", path, err)
		return "", "", nil
	}

	if len(root) != 1 {
		fmt.Printf("Error parsing file; should contain a single JSON object { ... } in %s", path)
		return "", "", nil
	}

	var resourceName, resourceId string
	for name, raw := range root {
		var idObj JSONID
		if err := json.Unmarshal(raw, &idObj); err != nil {
			fmt.Printf("Error parsing file; id not found in %s: %v", path, err)
			return "", "", nil
		}
		resourceName = name
		resourceId = idObj.Value
	}

	if resourceName == "" || resourceId == "" {
		fmt.Printf("Error parsing file: ResourceName or Id missing, %s", path)
		return "", "", nil
	}

	var resourceData map[string]interface{}
	if err := json.Unmarshal(data, &resourceData); err != nil {
		fmt.Printf("Error reading JSON as map for %s, %v", path, err)
		return "", "", nil
	}

	return resourceName, resourceId, resourceData
}

func saveAsJSON(databaseResourcePath string, resourceData map[string]interface{}) {
	jsonData, err := json.MarshalIndent(resourceData, "", "  ")
	if err != nil {
		fmt.Printf("Error converting resource to JSON %s: %v", databaseResourcePath, err)
		return
	}

	if err := ioutil.WriteFile(databaseResourcePath, jsonData, 0644); err != nil {
		fmt.Printf("Error writing resource to file %s: %v", databaseResourcePath, err)
		return
	}

	fmt.Printf("Saved: %s\n", databaseResourcePath)
}
