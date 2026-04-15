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

// allSpecs defines every class and spec in Midnight Season 1
var allSpecs = []struct {
	Class string
	Specs []string
}{
	{"DEATHKNIGHT", []string{"Blood", "Frost", "Unholy"}},
	{"DEMONHUNTER", []string{"Havoc", "Vengeance", "Devourer"}},
	{"DRUID", []string{"Balance", "Feral", "Guardian", "Restoration"}},
	{"EVOKER", []string{"Augmentation", "Devastation", "Preservation"}},
	{"HUNTER", []string{"BeastMastery", "Marksmanship", "Survival"}},
	{"MAGE", []string{"Arcane", "Fire", "Frost"}},
	{"MONK", []string{"Brewmaster", "Mistweaver", "Windwalker"}},
	{"PALADIN", []string{"Holy", "Protection", "Retribution"}},
	{"PRIEST", []string{"Discipline", "Holy", "Shadow"}},
	{"ROGUE", []string{"Assassination", "Outlaw", "Subtlety"}},
	{"SHAMAN", []string{"Elemental", "Enhancement", "Restoration"}},
	{"WARLOCK", []string{"Affliction", "Demonology", "Destruction"}},
	{"WARRIOR", []string{"Arms", "Fury", "Protection"}},
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
	case "init-specs":
		if err := initSpecs(); err != nil {
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
	fmt.Println("  go run . generate      Read data/*.json and generate GearPath_Data.lua")
	fmt.Println("  go run . init-specs    Create skeleton JSON files for all classes and specs")
}

func generate() error {
	jsonFiles, err := filepath.Glob("data/*.json")
	if err != nil {
		return fmt.Errorf("could not read data directory: %w", err)
	}
	if len(jsonFiles) == 0 {
		return fmt.Errorf("no JSON files found in data/")
	}

	sort.Strings(jsonFiles)

	var specs []SpecData
	for _, file := range jsonFiles {
		spec, err := parseSpecFile(file)
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
		// Skip empty specs — no point generating empty tables
		if len(spec.Items) == 0 {
			fmt.Printf("  Skipping (empty): %s %s\n", spec.Class, spec.Spec)
			continue
		}
		sort.Slice(spec.Items, func(i, j int) bool {
			return spec.Items[i].SlotID < spec.Items[j].SlotID
		})
		specs = append(specs, spec)
		fmt.Printf("  Loaded: %s %s (%d items)\n", spec.Class, spec.Spec, len(spec.Items))
	}

	if len(specs) == 0 {
		return fmt.Errorf("no specs with items found — nothing to generate")
	}

	tmpl, err := template.ParseFiles("templates/GearPath_Data.lua.tmpl")
	if err != nil {
		return fmt.Errorf("could not load template: %w", err)
	}

	outputPath := filepath.Join("..", "Data", "GearPath_Data.lua")
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer outFile.Close()

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

func initSpecs() error {
	today := time.Now().Format("2006-01-02")
	created := 0
	skipped := 0

	for _, class := range allSpecs {
		for _, spec := range class.Specs {
			filename := fmt.Sprintf("data/%s_%s.json", class.Class, spec)

			// Don't overwrite existing files
			if _, err := os.Stat(filename); err == nil {
				fmt.Printf("  Skipping (exists): %s\n", filename)
				skipped++
				continue
			}

			skeleton := SpecData{
				Class:   class.Class,
				Spec:    spec,
				Season:  "Midnight S1",
				Updated: today,
				Items:   []ItemData{},
			}

			data, err := json.MarshalIndent(skeleton, "", "  ")
			if err != nil {
				return fmt.Errorf("could not marshal %s %s: %w", class.Class, spec, err)
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return fmt.Errorf("could not write %s: %w", filename, err)
			}

			fmt.Printf("  Created: %s\n", filename)
			created++
		}
	}

	fmt.Printf("\nDone. Created: %d, Skipped: %d\n", created, skipped)
	return nil
}