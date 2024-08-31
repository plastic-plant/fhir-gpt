#!/usr/bin/env dotnet-script
// dotnet tool install -g dotnet-script
// dotnet script import.csx -- ../examples ../database

#r "nuget: Hl7.Fhir.R4, 5.9.1"

using Hl7.Fhir.Model;
using Hl7.Fhir.Serialization;
using System;

if (Args.Count < 2)
{
	Console.WriteLine("Usage:   import <examples folder> <database folder>");
	Console.WriteLine("Example: import ../examples ../database");
	Console.WriteLine("Example: dotnet run import.cs ../examples ../database");
	Console.WriteLine("Example: dotnet script import.cs -- ../examples ../database");
	Console.WriteLine();
	return;
}

var examplesFolder = Path.GetFullPath(Args[0]);
var databaseFolder = Path.GetFullPath(Args[1]);

if (!Directory.Exists(examplesFolder))
{
	Console.WriteLine($"The folder {examplesFolder} does not exist.");
	return;
}

string[] files = Directory.GetFiles(examplesFolder);
var parser = new FhirXmlParser();
var serializationSettings = new FhirJsonSerializationSettings();
serializationSettings.Pretty = true;

foreach (string file in files)
{
	var path = Path.GetFullPath(file);
	try
	{
		Resource resource = parser.Parse<Resource>(File.ReadAllText(path));
		if (string.IsNullOrWhiteSpace(resource.TypeName) || string.IsNullOrWhiteSpace(resource.Id))
		{
			Console.WriteLine($"Error: Missing root element name or id in file {path}");
		}

		var resourceFolder = Path.Combine(databaseFolder, resource.TypeName);
		Directory.CreateDirectory(resourceFolder);

		var resourceFile = Path.Combine(resourceFolder, $"{resource.Id}.json");
		var json = resource.ToJson(serializationSettings);
		File.WriteAllText(resourceFile, json);
		Console.WriteLine($"Successfully saved JSON file {resourceFile}");

	}
	catch (Exception ex)
	{
		Console.WriteLine($"Error processing {path}: {ex.Message}");
	}
}
