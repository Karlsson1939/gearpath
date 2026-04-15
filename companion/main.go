package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"
)

// SpecData mirrors the JSON file structure
type SpecData struct {
	Class   string     `json:"class"`
	Spec    string     `json:"spec"`
	Season  string     `json:"season"`
	Updated string     `json:"updated"`
	Items   []ItemData `json:"items"`
}

type ItemData struct {
	SlotID     int     `json:"slotID"`
	ItemID     int     `json:"itemID"`
	ItemName   string  `json:"itemName"`
	Source     string  `json:"source"`
	SourceType string  `json:"sourceType"`
	SourceName string  `json:"sourceName"`
	BossName   *string `json:"bossName"`
	Ilvl       int     `json:"ilvl"`
	Priority   int     `json:"priority"`
	IsTier     bool    `json:"isTier"`
}

// TemplateData is passed to the Lua template
type TemplateData struct {
	Generated string
	Specs     []SpecData
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "generate":
		if err := generate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("GearPath Companion App")
	fmt.Println("Usage:")
	fmt.Println("  go run . generate    Read data/*.json and generate GearPath_Data.lua")
}

func generate() error {
	// Find all JSON files in data/
	jsonFiles, err := filepath.Glob("data/*.json")
	if err != nil {
		return fmt.Errorf("could not read data directory: %w", err)
	}
	if len(jsonFiles) == 0 {
		return fmt.Errorf("no JSON files found in data/")
	}

	// Sort for deterministic output
	sort.Strings(jsonFiles)

	// Parse each JSON file
	var specs []SpecData
	for _, file := range jsonFiles {
		spec, err := parseSpecFile(file)
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
		// Sort items by slotID for clean output
		sort.Slice(spec.Items, func(i, j int) bool {
			return spec.Items[i].SlotID < spec.Items[j].SlotID
		})
		specs = append(specs, spec)
		fmt.Printf("  Loaded: %s %s (%d items)\n", spec.Class, spec.Spec, len(spec.Items))
	}

	// Load template
	tmpl, err := template.ParseFiles("templates/GearPath_Data.lua.tmpl")
	if err != nil {
		return fmt.Errorf("could not load template: %w", err)
	}

	// Output path — write directly to the addon Data folder
	outputPath := filepath.Join("..", "Data", "GearPath_Data.lua")
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer outFile.Close()

	// Execute template
	data := TemplateData{
		Generated: time.Now().Format("2006-01-02 15:04:05"),
		Specs:     specs,
	}
	if err := tmpl.Execute(outFile, data); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	fmt.Printf("\nGenerated: %s\n", outputPath)
	fmt.Printf("Total specs: %d\n", len(specs))
	return nil
}

func parseSpecFile(path string) (SpecData, error) {
	f, err := os.Open(path)
	if err != nil {
		return SpecData{}, err
	}
	defer f.Close()

	var spec SpecData
	if err := json.NewDecoder(f).Decode(&spec); err != nil {
		return SpecData{}, err
	}
	return spec, nil
}