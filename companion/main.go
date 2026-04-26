package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/chromedp/chromedp"
)

// ============================================================
// Data structures
// ============================================================

type SpecData struct {
	Class   string     `json:"class"`
	Spec    string     `json:"spec"`
	HeroKey string     `json:"heroKey"` // "any" or specific hero talent name (e.g. "Deathbringer")
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
	HeroSpec   string  `json:"heroSpec,omitempty"` // "" = applies to all; else specific hero talent
}

type TemplateData struct {
	Generated string
	Groups    []SpecGroup
}

// SpecGroup aggregates all hero lists for one class+spec so the template can
// emit a single Lua block per spec with nested hero keys.
type SpecGroup struct {
	Class  string
	Spec   string
	Season string
	Heroes []HeroGroup
}

type HeroGroup struct {
	HeroKey string
	Updated string
	Items   []ItemData
}

// ============================================================
// Blizzard API structures
// ============================================================

type BlizzardToken struct {
	AccessToken string `json:"access_token"`
}

type BlizzardItem struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Level         int    `json:"level"`
	InventoryType struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"inventory_type"`
	PreviewItem struct {
		Source struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"source"`
	} `json:"preview_item"`
}

type JournalInstanceIndex struct {
	Instances []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"instances"`
}

type JournalInstance struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	InstanceType struct {
		Type string `json:"type"`
	} `json:"instance_type"`
	Encounters []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"encounters"`
}

type ItemSource struct {
	ItemName   string `json:"itemName"`
	BossName   string `json:"bossName"`
	SourceName string `json:"sourceName"`
	SourceType string `json:"sourceType"`
	Source     string `json:"source"`
}

type SourceMapCache struct {
	Season string             `json:"season"`
	Built  string             `json:"built"`
	Items  map[int]ItemSource `json:"items"`
	Names  map[string]int     `json:"names"`
}

const sourceMapCachePath = "data/sourcemap_midnight_s1.json"

// ============================================================
// Season 1 instance whitelist
// ============================================================

var season1Instances = []string{
	"Magisters' Terrace",
	"Maisara Caverns",
	"Nexus-Point Xenas",
	"Windrunner Spire",
	"Algeth'ar Academy",
	"Seat of the Triumvirate",
	"Skyreach",
	"Pit of Saron",
	"The Voidspire",
	"The Dreamrift",
	"March on Quel'Danas",
}

// ============================================================
// Icy Veins URL slugs
// ============================================================

var icyVeinsSlugs = map[string]string{
	"DEATHKNIGHT_Blood":     "blood-death-knight-pve-tank",
	"DEATHKNIGHT_Frost":     "frost-death-knight-pve-dps",
	"DEATHKNIGHT_Unholy":    "unholy-death-knight-pve-dps",
	"DEMONHUNTER_Havoc":     "havoc-demon-hunter-pve-dps",
	"DEMONHUNTER_Vengeance": "vengeance-demon-hunter-pve-tank",
	"DEMONHUNTER_Devourer":  "devourer-demon-hunter-pve-dps",
	"DRUID_Balance":         "balance-druid-pve-dps",
	"DRUID_Feral":           "feral-druid-pve-dps",
	"DRUID_Guardian":        "guardian-druid-pve-tank",
	"DRUID_Restoration":     "restoration-druid-pve-healing",
	"EVOKER_Augmentation":   "augmentation-evoker-pve-dps",
	"EVOKER_Devastation":    "devastation-evoker-pve-dps",
	"EVOKER_Preservation":   "preservation-evoker-pve-healing",
	"HUNTER_BeastMastery":   "beast-mastery-hunter-pve-dps",
	"HUNTER_Marksmanship":   "marksmanship-hunter-pve-dps",
	"HUNTER_Survival":       "survival-hunter-pve-dps",
	"MAGE_Arcane":           "arcane-mage-pve-dps",
	"MAGE_Fire":             "fire-mage-pve-dps",
	"MAGE_Frost":            "frost-mage-pve-dps",
	"MONK_Brewmaster":       "brewmaster-monk-pve-tank",
	"MONK_Mistweaver":       "mistweaver-monk-pve-healing",
	"MONK_Windwalker":       "windwalker-monk-pve-dps",
	"PALADIN_Holy":          "holy-paladin-pve-healing",
	"PALADIN_Protection":    "protection-paladin-pve-tank",
	"PALADIN_Retribution":   "retribution-paladin-pve-dps",
	"PRIEST_Discipline":     "discipline-priest-pve-healing",
	"PRIEST_Holy":           "holy-priest-pve-healing",
	"PRIEST_Shadow":         "shadow-priest-pve-dps",
	"ROGUE_Assassination":   "assassination-rogue-pve-dps",
	"ROGUE_Outlaw":          "outlaw-rogue-pve-dps",
	"ROGUE_Subtlety":        "subtlety-rogue-pve-dps",
	"SHAMAN_Elemental":      "elemental-shaman-pve-dps",
	"SHAMAN_Enhancement":    "enhancement-shaman-pve-dps",
	"SHAMAN_Restoration":    "restoration-shaman-pve-healing",
	"WARLOCK_Affliction":    "affliction-warlock-pve-dps",
	"WARLOCK_Demonology":    "demonology-warlock-pve-dps",
	"WARLOCK_Destruction":   "destruction-warlock-pve-dps",
	"WARRIOR_Arms":          "arms-warrior-pve-dps",
	"WARRIOR_Fury":          "fury-warrior-pve-dps",
	"WARRIOR_Protection":    "protection-warrior-pve-tank",
}

// heroTalentSplits declares specs where Icy Veins publishes fundamentally
// different BiS lists per hero talent. These specs get one JSON file per
// hero talent. The order of hero keys in the slice is the order we expect
// them to appear on Icy Veins' page (used by the heading matcher as a hint
// for error messages only — the actual matching is by heading text).
var heroTalentSplits = map[string][]string{
	"DEATHKNIGHT_Blood": {"San'layn", "Deathbringer"},
}

// heroGuideTab describes a hero-talent tab on an Icy Veins guide page.
type heroGuideTab struct {
	Key                string         // normalized key matching Detection.lua's normalizeKey
	DisplayName        string         // human-readable name
	StatSelector       string         // CSS selector on stat page; empty = single-read
	ConsumHeroSelector string         // CSS selector to switch hero on consumables page
	ConsumSubTabs      []consumSubTab // sub-tabs within this hero's consumable section
}

// consumSubTab describes a sub-tab within a hero talent's consumable section.
// Content determines which parser runs on the page text after clicking.
type consumSubTab struct {
	Selector string // CSS selector for the sub-tab button
	Content  string // "gems", "enchants", or "consumables"
}

// heroGuideSpec groups the hero talent tabs for a spec.
type heroGuideSpec struct {
	Tabs                 []heroGuideTab
	FallbackSelector     string // if non-empty, click this on stat page and store as _any
	ConsumSubTabsEnabled bool   // when true, use nested-tab consumable scraping per hero
}

// heroGuideSplits declares specs where Icy Veins publishes per-hero-talent
// content behind tab UIs on stat or consumable pages. Tabs with empty
// StatSelector use a single stat-page read (both hero talents' stats visible).
// When ConsumSubTabsEnabled is true, the scraper clicks per-hero sub-tabs
// on the consumables page. Otherwise consumables are parsed once and shared.
//
// Inline prose consumable differences (e.g., Blood DK Deathbringer vs San'layn
// flask) are not captured; see Icy Veins for hero-specific consumable nuance.
var heroGuideSplits = map[string]heroGuideSpec{
	"DEATHKNIGHT_Blood": {
		Tabs: []heroGuideTab{
			{Key: "Deathbringer", DisplayName: "Deathbringer", StatSelector: "#area_1_button"},
			{Key: "San'layn", DisplayName: "San'layn", StatSelector: "#area_2_button"},
		},
	},
	"PALADIN_Holy": {
		FallbackSelector: "#area_1_button", // "General Stat Priority" — _any fallback
		Tabs: []heroGuideTab{
			{Key: "HeraldoftheSun", DisplayName: "Herald of the Sun", StatSelector: "#area_2_button"},
			{Key: "Lightsmith", DisplayName: "Lightsmith", StatSelector: "#area_3_button"},
		},
	},
	"PRIEST_Discipline": {
		ConsumSubTabsEnabled: true,
		Tabs: []heroGuideTab{
			{
				Key:                "Oracle",
				DisplayName:        "Oracle",
				ConsumHeroSelector: "#rotation_switch_oracle",
				ConsumSubTabs: []consumSubTab{
					{Selector: "#area_1_button", Content: "gems"},
					{Selector: "#area_2_button", Content: "enchants"},
					{Selector: "#area_3_button", Content: "consumables"},
				},
			},
			{
				Key:                "Voidweaver",
				DisplayName:        "Voidweaver",
				ConsumHeroSelector: "#rotation_switch_voidweaver",
				ConsumSubTabs: []consumSubTab{
					{Selector: "#area_4_button", Content: "gems"},
					{Selector: "#area_5_button", Content: "enchants"},
					{Selector: "#area_6_button", Content: "consumables"},
				},
			},
		},
	},
	"PRIEST_Shadow": {
		Tabs: []heroGuideTab{
			{Key: "Archon", DisplayName: "Archon", StatSelector: "#rotation_switch_archon"},
			{Key: "Voidweaver", DisplayName: "Voidweaver", StatSelector: "#rotation_switch_voidweaver"},
		},
	},
	"SHAMAN_Enhancement": {
		Tabs: []heroGuideTab{
			{Key: "Stormbringer", DisplayName: "Stormbringer", StatSelector: "#area_1_button"},
			{Key: "Totemic", DisplayName: "Totemic", StatSelector: "#area_2_button"},
		},
	},
}

// ============================================================
// Icy Veins slot parsing maps
// ============================================================

var icyVeinsSlotMap = map[string]int{
	// Main hand / weapons
	"weapon":                      16,
	"weapons":                     16,
	"main hand":                   16,
	"mainhand weapon":             16,
	"one-handed weapon":           16,
	"1h weapon":                   16,
	"1h":                          16,
	"2h weapon":                   16,
	"2h":                          16,
	"weapon (2h)":                 16,
	"weapon (two-hand)":           16,
	"weapon (main-hand/off-hand)": 16,
	"weapon (staff)":              16,
	"mainhand 1h weapon":          16,
	"offhand 1h weapon":           17,
	"sentinel weapon":             16,
	"weapon main-hand":            16,
	"weapons (dual wield)":        16,
	"two-handed weapon":           16,
	"two-handed mace":             16,
	"staff":                       16,

	// Off hand
	"off hand":        17,
	"off-hand":        17,
	"offhand":         17,
	"offhand weapon":  17,
	"off-hand weapon": 17,
	"weapon off-hand": 17,
	"shield":          17,

	// Armour
	"helm":      1,
	"helmet":    1,
	"head":      1,
	"neck":      2,
	"necklace":  2,
	"shoulder":  3,
	"shoulders": 3,
	"cloak":     15,
	"cape":      15,
	"back":      15,
	"chest":     5,
	"bracers":   9,
	"wrist":     9,
	"wrists":    9,
	"gloves":    10,
	"glove":     10,
	"hands":     10,
	"belt":      6,
	"waist":     6,
	"legs":      7,
	"leggings":  7,
	"boots":     8,
	"feet":      8,

	// Rings — numbered
	"ring #1":   11,
	"ring #2":   12,
	"ring 1":    11,
	"ring 2":    12,
	"finger #1": 11,
	"finger #2": 12,
	"finger 1":  11,
	"finger 2":  12,

	// Rings — unnumbered / plural
	"ring":  11,
	"rings": 11,

	// Trinkets — numbered
	"trinket #1": 13,
	"trinket #2": 14,
	"trinket 1":  13,
	"trinket 2":  14,

	// Trinkets — unnumbered / plural
	"trinket":      13,
	"trinkets":     13,
	"top trinkets": 13,
}

var pairFallback = map[int]int{
	11: 12,
	13: 14,
}

var multiItemSlots = map[string]bool{
	"rings":        true,
	"top trinkets": true,
	"trinkets":     true,
	"trinket":      true,
}

// ============================================================
// Slot definitions
// ============================================================

var slots = []struct {
	ID   int
	Name string
}{
	{1, "Head"},
	{2, "Neck"},
	{3, "Shoulders"},
	{5, "Chest"},
	{6, "Waist"},
	{7, "Legs"},
	{8, "Feet"},
	{9, "Wrists"},
	{10, "Hands"},
	{11, "Finger 1"},
	{12, "Finger 2"},
	{13, "Trinket 1"},
	{14, "Trinket 2"},
	{15, "Back"},
	{16, "Main Hand"},
	{17, "Off Hand"},
}

var allSpecs = []struct {
	Class string
	Spec  string
}{
	{"DEATHKNIGHT", "Blood"},
	{"DEATHKNIGHT", "Frost"},
	{"DEATHKNIGHT", "Unholy"},
	{"DEMONHUNTER", "Havoc"},
	{"DEMONHUNTER", "Vengeance"},
	{"DEMONHUNTER", "Devourer"},
	{"DRUID", "Balance"},
	{"DRUID", "Feral"},
	{"DRUID", "Guardian"},
	{"DRUID", "Restoration"},
	{"EVOKER", "Augmentation"},
	{"EVOKER", "Devastation"},
	{"EVOKER", "Preservation"},
	{"HUNTER", "BeastMastery"},
	{"HUNTER", "Marksmanship"},
	{"HUNTER", "Survival"},
	{"MAGE", "Arcane"},
	{"MAGE", "Fire"},
	{"MAGE", "Frost"},
	{"MONK", "Brewmaster"},
	{"MONK", "Mistweaver"},
	{"MONK", "Windwalker"},
	{"PALADIN", "Holy"},
	{"PALADIN", "Protection"},
	{"PALADIN", "Retribution"},
	{"PRIEST", "Discipline"},
	{"PRIEST", "Holy"},
	{"PRIEST", "Shadow"},
	{"ROGUE", "Assassination"},
	{"ROGUE", "Outlaw"},
	{"ROGUE", "Subtlety"},
	{"SHAMAN", "Elemental"},
	{"SHAMAN", "Enhancement"},
	{"SHAMAN", "Restoration"},
	{"WARLOCK", "Affliction"},
	{"WARLOCK", "Demonology"},
	{"WARLOCK", "Destruction"},
	{"WARRIOR", "Arms"},
	{"WARRIOR", "Fury"},
	{"WARRIOR", "Protection"},
}

// ============================================================
// Blizzard client
// ============================================================

type BlizzardClient struct {
	token      string
	httpClient *http.Client
	sourceMap  map[int]ItemSource
	nameMap    map[string]int
}

func newBlizzardClient() (*BlizzardClient, error) {
	clientID := os.Getenv("BLIZZARD_CLIENT_ID")
	clientSecret := os.Getenv("BLIZZARD_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("BLIZZARD_CLIENT_ID and BLIZZARD_CLIENT_SECRET must be set")
	}

	c := &BlizzardClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		sourceMap:  map[int]ItemSource{},
		nameMap:    map[string]int{},
	}

	if err := c.authenticate(clientID, clientSecret); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *BlizzardClient) authenticate(clientID, clientSecret string) error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, _ := http.NewRequest("POST", "https://oauth.battle.net/token",
		bytes.NewBufferString(data.Encode()))
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	var token BlizzardToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("auth decode failed: %w", err)
	}
	if token.AccessToken == "" {
		return fmt.Errorf("received empty access token — check credentials")
	}

	c.token = token.AccessToken
	return nil
}

func (c *BlizzardClient) get(path string, out interface{}) error {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("not found: %s", path)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *BlizzardClient) getRaw(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func extractItemIDsFromEncounter(raw []byte) []int {
	var result struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}

	var ids []int
	for _, entry := range result.Items {
		var shapeA struct {
			Item struct {
				ID int `json:"id"`
			} `json:"item"`
		}
		if err := json.Unmarshal(entry, &shapeA); err == nil && shapeA.Item.ID > 0 {
			ids = append(ids, shapeA.Item.ID)
			continue
		}
		var shapeB struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(entry, &shapeB); err == nil && shapeB.ID > 0 {
			ids = append(ids, shapeB.ID)
		}
	}
	return ids
}

func (c *BlizzardClient) loadOrBuildSourceMap() error {
	if raw, err := os.ReadFile(sourceMapCachePath); err == nil {
		var cache SourceMapCache
		if err := json.Unmarshal(raw, &cache); err == nil && len(cache.Items) > 0 {
			c.sourceMap = cache.Items
			c.nameMap = cache.Names
			fmt.Printf("Source map loaded from cache (%d items, built %s)\n\n",
				len(c.sourceMap), cache.Built)
			return nil
		}
		fmt.Println("Cache found but could not be parsed — rebuilding...")
	}

	fmt.Println("Building item source map from Blizzard journal...")

	var index JournalInstanceIndex
	if err := c.get(
		"https://us.api.blizzard.com/data/wow/journal-instance/index?namespace=static-us&locale=en_US",
		&index,
	); err != nil {
		return fmt.Errorf("failed to fetch journal index: %w", err)
	}

	whitelist := make(map[string]bool)
	for _, name := range season1Instances {
		whitelist[name] = true
	}

	matched := 0
	for _, inst := range index.Instances {
		if !whitelist[inst.Name] {
			continue
		}

		matched++
		fmt.Printf("  [%d/%d] %s... ", matched, len(season1Instances), inst.Name)

		var instance JournalInstance
		if err := c.get(fmt.Sprintf(
			"https://us.api.blizzard.com/data/wow/journal-instance/%d?namespace=static-us&locale=en_US",
			inst.ID,
		), &instance); err != nil {
			fmt.Printf("FAILED (%v)\n", err)
			continue
		}

		sourceType := "DUNGEON"
		if instance.InstanceType.Type == "RAID" {
			sourceType = "RAID"
		}

		itemCount := 0
		for _, enc := range instance.Encounters {
			raw, err := c.getRaw(fmt.Sprintf(
				"https://us.api.blizzard.com/data/wow/journal-encounter/%d?namespace=static-us&locale=en_US",
				enc.ID,
			))
			if err != nil {
				continue
			}

			ids := extractItemIDsFromEncounter(raw)
			for _, id := range ids {
				c.sourceMap[id] = ItemSource{
					BossName:   enc.Name,
					SourceName: instance.Name,
					SourceType: sourceType,
					Source:     slugify(instance.Name),
				}
				itemCount++
			}

			time.Sleep(100 * time.Millisecond)
		}

		fmt.Printf("%d items\n", itemCount)
	}

	if matched < len(season1Instances) {
		fmt.Printf("  Warning: only matched %d/%d instances\n", matched, len(season1Instances))
	}

	fmt.Printf("Source map complete: %d items indexed\n", len(c.sourceMap))

	fmt.Printf("Building name index for %d journal items", len(c.sourceMap))
	count := 0
	for id := range c.sourceMap {
		item, err := c.fetchItem(id)
		if err != nil {
			continue
		}
		lower := strings.ToLower(item.Name)
		c.nameMap[lower] = id
		if src, ok := c.sourceMap[id]; ok {
			src.ItemName = item.Name
			c.sourceMap[id] = src
		}
		count++
		if count%50 == 0 {
			fmt.Print(".")
		}
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Printf(" done (%d names)\n", count)

	cache := SourceMapCache{
		Season: "Midnight S1",
		Built:  time.Now().Format("2006-01-02"),
		Items:  c.sourceMap,
		Names:  c.nameMap,
	}
	if data, err := json.MarshalIndent(cache, "", "  "); err == nil {
		if err := os.WriteFile(sourceMapCachePath, data, 0644); err == nil {
			fmt.Printf("Source map cached to %s\n", sourceMapCachePath)
		}
	}

	fmt.Println()
	return nil
}

func (c *BlizzardClient) fetchItem(itemID int) (*BlizzardItem, error) {
	var item BlizzardItem
	if err := c.get(fmt.Sprintf(
		"https://us.api.blizzard.com/data/wow/item/%d?namespace=static-us&locale=en_US",
		itemID,
	), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func slugify(s string) string {
	return strings.NewReplacer(" ", "", "'", "", "-", "", ".", "").Replace(strings.ToLower(s))
}

// slugifyHero normalises a hero talent name for use as a filename component.
// Apostrophes and spaces are stripped; the result is filesystem-safe.
//   "any"         → "any"
//   "San'layn"    → "Sanlayn"
//   "Pack Leader" → "PackLeader"
func slugifyHero(hero string) string {
	return strings.NewReplacer("'", "", " ", "", "-", "").Replace(hero)
}

// buildDataFilename produces the canonical data JSON path for a spec+hero combo.
func buildDataFilename(class, spec, heroKey string) string {
	return fmt.Sprintf("data/%s_%s_%s.json", class, spec, slugifyHero(heroKey))
}

// ============================================================
// Main
// ============================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "generate":
			if err := generate(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "init-specs":
			if err := initSpecs(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "list-instances":
			if err := listInstances(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "debug-encounter":
			if err := debugEncounter(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "refresh-sourcemap":
			if err := os.Remove(sourceMapCachePath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error removing cache: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Source map cache cleared.")
			fmt.Println("Run 'go run . scrape' to rebuild from the Blizzard API.")
			return
		case "scrape":
			target := ""
			if len(os.Args) > 2 {
				target = os.Args[2]
			}
			fmt.Print("Authenticating with Blizzard API... ")
			blizzard, err := newBlizzardClient()
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAILED\n%v\n", err)
				os.Exit(1)
			}
			fmt.Println("OK")
			if err := blizzard.loadOrBuildSourceMap(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: source map failed: %v\n", err)
			}
			if err := scrapeAll(target, blizzard); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "scrape-guide":
			target := ""
			if len(os.Args) > 2 {
				target = os.Args[2]
			}
			if err := scrapeGuideAll(target); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "generate-guide":
			if err := generateGuide(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "validate-items":
			strict := len(os.Args) > 2 && os.Args[2] == "--strict"
			if err := validateItems(strict); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "resolve-items":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: go run . resolve-items \"Name One\" \"Name Two\" ...\n")
				os.Exit(1)
			}
			if err := resolveItemsScan(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// Default: interactive editor
	fmt.Println("╔══════════════════════════════╗")
	fmt.Println("║   GearPath Data Editor       ║")
	fmt.Println("║   Midnight Season 1          ║")
	fmt.Println("╚══════════════════════════════╝")
	fmt.Println()

	fmt.Print("Authenticating with Blizzard API... ")
	blizzard, err := newBlizzardClient()
	if err != nil {
		fmt.Printf("FAILED\n%v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	if err := blizzard.loadOrBuildSourceMap(); err != nil {
		fmt.Printf("Warning: source map failed (%v)\n", err)
		fmt.Println("Continuing — source info will need to be entered manually.")
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		s := selectSpec(reader)
		if s == nil {
			fmt.Println("\nGoodbye!")
			return
		}
		editSpec(reader, blizzard, s.Class, s.Spec, s.HeroKey)
	}
}

// ============================================================
// Debug encounter
// ============================================================

func debugEncounter() error {
	clientID := os.Getenv("BLIZZARD_CLIENT_ID")
	clientSecret := os.Getenv("BLIZZARD_CLIENT_SECRET")
	c := &BlizzardClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		sourceMap:  map[int]ItemSource{},
		nameMap:    map[string]int{},
	}
	if err := c.authenticate(clientID, clientSecret); err != nil {
		return err
	}

	var index JournalInstanceIndex
	if err := c.get(
		"https://us.api.blizzard.com/data/wow/journal-instance/index?namespace=static-us&locale=en_US",
		&index,
	); err != nil {
		return err
	}

	var voidspireID int
	for _, inst := range index.Instances {
		if inst.Name == "The Voidspire" {
			voidspireID = inst.ID
			break
		}
	}
	if voidspireID == 0 {
		return fmt.Errorf("The Voidspire not found in index")
	}

	var instance JournalInstance
	if err := c.get(fmt.Sprintf(
		"https://us.api.blizzard.com/data/wow/journal-instance/%d?namespace=static-us&locale=en_US",
		voidspireID,
	), &instance); err != nil {
		return err
	}

	fmt.Printf("The Voidspire encounters: %d\n", len(instance.Encounters))
	for _, enc := range instance.Encounters {
		fmt.Printf("  Boss: %s (ID %d)\n", enc.Name, enc.ID)
	}

	if len(instance.Encounters) == 0 {
		return fmt.Errorf("no encounters found")
	}

	enc := instance.Encounters[0]
	fmt.Printf("\nRaw JSON for encounter: %s\n", enc.Name)
	raw, err := c.getRaw(fmt.Sprintf(
		"https://us.api.blizzard.com/data/wow/journal-encounter/%d?namespace=static-us&locale=en_US",
		enc.ID,
	))
	if err != nil {
		return err
	}

	var pretty interface{}
	json.Unmarshal(raw, &pretty)
	out, _ := json.MarshalIndent(pretty, "", "  ")

	os.WriteFile("/tmp/encounter_debug.json", out, 0644)
	fmt.Println("Full response written to /tmp/encounter_debug.json")
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i >= 60 {
			fmt.Printf("... (%d more lines)\n", len(lines)-60)
			break
		}
		fmt.Println(line)
	}

	return nil
}

// ============================================================
// Scrape
// ============================================================

// scrapeTarget is one unit of scrape work: a spec + hero key combination.
// For non-split specs, HeroKey is "any". For split specs (e.g. Blood DK),
// there's one target per hero talent.
type scrapeTarget struct {
	Class   string
	Spec    string
	HeroKey string // "any" or specific hero talent name
}

// expandScrapeTargets produces the full list of {class, spec, heroKey} combos
// to scrape. Common specs → one target with heroKey="any". Split specs →
// one target per hero talent.
func expandScrapeTargets() []scrapeTarget {
	var targets []scrapeTarget
	for _, s := range allSpecs {
		specKey := s.Class + "_" + s.Spec
		if heroes, split := heroTalentSplits[specKey]; split {
			for _, hero := range heroes {
				targets = append(targets, scrapeTarget{
					Class:   s.Class,
					Spec:    s.Spec,
					HeroKey: hero,
				})
			}
			continue
		}
		targets = append(targets, scrapeTarget{
			Class:   s.Class,
			Spec:    s.Spec,
			HeroKey: "any",
		})
	}
	return targets
}

func scrapeAll(target string, blizzard *BlizzardClient) error {
	fmt.Println("Starting headless browser...")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	fmt.Println("Browser ready.")
	fmt.Println()

	today := time.Now().Format("2006-01-02")
	fetched, failed, skipped := 0, 0, 0

	// For split specs we cache the scraped page so we only hit Icy Veins once
	// per spec even though we emit multiple targets from it.
	type pageCache struct {
		tables []scrapedTable
	}
	splitPageCache := map[string]*pageCache{}

	targets := expandScrapeTargets()

	for _, t := range targets {
		specKey := t.Class + "_" + t.Spec
		fullKey := specKey + "_" + t.HeroKey

		// Target filter accepts "CLASS_Spec" (all heroes) or "CLASS_Spec_Hero"
		if target != "" && target != specKey && target != fullKey {
			continue
		}

		slug, ok := icyVeinsSlugs[specKey]
		if !ok {
			fmt.Printf("  No slug for %s — skipping\n", specKey)
			skipped++
			continue
		}

		label := fmt.Sprintf("%s %s", t.Class, t.Spec)
		if t.HeroKey != "any" {
			label += " (" + t.HeroKey + ")"
		}
		fmt.Printf("Scraping %-45s ... ", label)

		pageURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s-gear-best-in-slot", slug)

		// Fetch tables (with headings) from the page. Cache for split specs.
		var tables []scrapedTable
		if cached, ok := splitPageCache[specKey]; ok {
			tables = cached.tables
		} else {
			var err error
			tables, err = fetchPageTables(browserCtx, pageURL)
			if err != nil {
				fmt.Printf("FAILED (browser: %v)\n", err)
				failed++
				continue
			}
			if _, isSplit := heroTalentSplits[specKey]; isSplit {
				splitPageCache[specKey] = &pageCache{tables: tables}
			}
		}

		// Pick the right table for this target.
		tableData, pickErr := pickTableForTarget(tables, t)
		if pickErr != nil {
			fmt.Printf("FAILED (%v)\n", pickErr)
			failed++
			continue
		}

		if strings.TrimSpace(tableData) == "" {
			fmt.Printf("FAILED (empty table)\n")
			failed++
			continue
		}

		items := parseIcyVeinsTable(tableData, blizzard)
		if len(items) == 0 {
			fmt.Printf("FAILED (no items parsed)\n")
			failed++
			continue
		}

		fmt.Printf("%d items saved\n", len(items))

		specData := SpecData{
			Class:   t.Class,
			Spec:    t.Spec,
			HeroKey: t.HeroKey,
			Season:  "Midnight S1",
			Updated: today,
			Items:   items,
		}

		filename := buildDataFilename(t.Class, t.Spec, t.HeroKey)
		data, _ := json.MarshalIndent(specData, "", "  ")
		if err := os.WriteFile(filename, data, 0644); err != nil {
			fmt.Printf("FAILED (write: %v)\n", err)
			failed++
			continue
		}

		fetched++
		// Only rate-limit between page fetches, not between cached-table reads.
		if _, wasCached := splitPageCache[specKey]; !wasCached || len(splitPageCache[specKey].tables) == 0 {
			time.Sleep(800 * time.Millisecond)
		}
	}

	fmt.Printf("\nDone. Scraped: %d, Failed: %d, Skipped: %d\n", fetched, failed, skipped)

	unresolved := findUnresolvedItems(today, target)
	if len(unresolved) > 0 {
		fmt.Printf("\nUnresolved items (itemID=0) — add to knownItems in main.go:\n")
		for _, u := range unresolved {
			fmt.Printf("  [%s] %s\n", u.slot, u.name)
		}
	} else {
		fmt.Println("\nAll items resolved. ✓")
	}

	return nil
}

// scrapedTable is a table from a BiS page, paired with contextual hints that
// help us identify which hero talent it belongs to.
//
// For Icy Veins BiS pages, each list lives inside a <div id="area_N"> and the
// header block above them (<div class="image_block_header">) contains the
// human-readable tab labels in matching order — so we capture both pieces
// and reconcile them.
type scrapedTable struct {
	AreaID string // e.g. "area_1" — the containing div's id, or "" for non-area tables
	Label  string // e.g. "BiS Raid (San'layn)" — tab label matched to AreaID, or ""
	TSV    string // tab-separated row data
}

// fetchPageTables navigates to pageURL, extracts every BiS table on the page,
// and pairs each with its area_N id and the corresponding tab label.
//
// The mapping is: each <div id="area_N"> contains one BiS table. The
// <div class="image_block_header"> that sits alongside those areas contains
// the tab labels in the same order — so area_1's label is the first label,
// area_2's is the second, etc.
func fetchPageTables(browserCtx context.Context, pageURL string) ([]scrapedTable, error) {
	pageCtx, pageCancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer pageCancel()

	var rawJSON string
	err := chromedp.Run(pageCtx,
		chromedp.Navigate(pageURL),
		chromedp.WaitVisible(`table tr`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`
			(function() {
				function tableToTSV(t) {
					return Array.from(t.querySelectorAll('tr'))
						.map(r => Array.from(r.querySelectorAll('td,th'))
							.map(c => c.innerText.trim().replace(/\n+/g, ' '))
							.join('\t'))
						.join('\n');
				}

				const results = [];

				// Primary path: gear BiS tables live inside <div id="area_N"> within
				// a <div class="image_block">. Each area has a matching tab button
				// <span id="area_N_button"> whose text is the tab's human-readable
				// label. We pair them by id, not by position — so reordering, extra
				// sections, or new blocks never cause mismatches.
				const blocks = document.querySelectorAll('div.image_block');
				blocks.forEach(block => {
					const areas = block.querySelectorAll('div[id^="area_"]');
					areas.forEach(area => {
						const table = area.querySelector('table');
						if (!table) return;

						// Find the matching button by id: area_N  →  area_N_button.
						const btn = block.querySelector('#' + area.id + '_button');
						const label = btn ? btn.innerText.trim() : '';

						results.push({
							areaId: area.id,
							label:  label,
							tsv:    tableToTSV(table),
						});
					});
				});

				// Fallback path: pages without the image_block/area_N structure
				// (i.e. every common spec) get their tables enumerated as-is, with
				// no areaId/label. This preserves the original behaviour for the
				// ~38 non-split specs.
				if (results.length === 0) {
					const loose = document.querySelectorAll('table');
					loose.forEach(t => {
						// Skip tables inside trinket ranking dropdowns — they're not BiS.
						if (t.closest('details.trinket-dropdown')) return;
						results.push({ areaId: '', label: '', tsv: tableToTSV(t) });
					});
				}

				return JSON.stringify(results);
			})()
		`, &rawJSON),
	)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		AreaID string `json:"areaId"`
		Label  string `json:"label"`
		TSV    string `json:"tsv"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return nil, fmt.Errorf("parse tables JSON: %w", err)
	}

	tables := make([]scrapedTable, 0, len(raw))
	for _, r := range raw {
		tables = append(tables, scrapedTable{
			AreaID: r.AreaID,
			Label:  r.Label,
			TSV:    r.TSV,
		})
	}
	return tables, nil
}

// pickTableForTarget selects the right table for a scrape target.
//
// Common specs (HeroKey="any"): returns the first non-empty table, which
// matches the original scraper behaviour. For pages with area_N blocks,
// this is area_1 (the first/default BiS list); for other pages, it's the
// first table on the page.
//
// Split specs: matches the hero talent name against tab labels (e.g.
// "BiS Raid (San'layn)") with apostrophe- and case-insensitive comparison.
// Fails loudly with the labels we did find, so a scraper break is easy to
// diagnose.
func pickTableForTarget(tables []scrapedTable, t scrapeTarget) (string, error) {
	if len(tables) == 0 {
		return "", fmt.Errorf("no tables on page")
	}

	if t.HeroKey == "any" {
		for _, tbl := range tables {
			if strings.TrimSpace(tbl.TSV) != "" {
				return tbl.TSV, nil
			}
		}
		return "", fmt.Errorf("all tables empty")
	}

	// Split spec: match against the tab label.
	// Normalise both sides: strip apostrophes, lowercase.
	normalise := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(s, "'", ""))
	}
	want := normalise(t.HeroKey)

	for _, tbl := range tables {
		if tbl.Label != "" && strings.Contains(normalise(tbl.Label), want) &&
			strings.TrimSpace(tbl.TSV) != "" {
			return tbl.TSV, nil
		}
	}

	// Nothing matched — show what we actually saw so the user can update
	// heroTalentSplits or the scraper if Icy Veins changed its markup.
	seen := make([]string, 0, len(tables))
	for _, tbl := range tables {
		if tbl.Label != "" {
			seen = append(seen, fmt.Sprintf("%s=%q", tbl.AreaID, tbl.Label))
		} else if tbl.AreaID != "" {
			seen = append(seen, fmt.Sprintf("%s=(unlabelled)", tbl.AreaID))
		}
	}
	return "", fmt.Errorf("no table found matching %q; saw: %s",
		t.HeroKey, strings.Join(seen, ", "))
}

// nameAliases maps Icy Veins display names to Blizzard API canonical names
// for items where the two sources disagree. Each value MUST exist as a key in
// knownItems — the validate-items command checks knownItems against the API,
// so aliases that point to non-existent keys would silently drop items.
var nameAliases = map[string]string{
	// Blizzard renamed this item via hotfix; Icy Veins still uses the old name.
	"Scabrous Zombie Leather Belt": "Scabrous Zombie Belt",
}

// parseIcyVeinsTable parses the tab-separated rows from a BiS table.
//
// Normal rows: SlotLabel \t ItemName \t Source
//
// Special cases handled:
//   - Multi-item cells (Rings, Top Trinkets): split on double-space, emit two items
//   - Weapons with (1H)/(OH) markers: extract main hand and off hand separately
//   - Pair slots (Ring, Trinkets) repeated unlabelled: use pairFallback
//   - Suffix stripping: "(TIER SET)", "(Crit/Haste)", " with ...", etc.
func parseIcyVeinsTable(tableData string, blizzard *BlizzardClient) []ItemData {
	lines := strings.Split(strings.TrimSpace(tableData), "\n")
	var items []ItemData
	seen := map[int]bool{}

	addItem := func(slotID int, itemName string) {
		// Resolve Icy Veins name variants to API canonical names.
		if canonical, ok := nameAliases[itemName]; ok {
			itemName = canonical
		}
		item := ItemData{
			SlotID:   slotID,
			ItemName: itemName,
			Priority: priorityForSlot(slotID),
			Ilvl:     252,
		}
		if id, ok := blizzard.nameMap[strings.ToLower(itemName)]; ok {
			item.ItemID = id
			if src, ok := blizzard.sourceMap[id]; ok {
				item.Source = src.Source
				item.SourceType = src.SourceType
				item.SourceName = src.SourceName
				boss := src.BossName
				item.BossName = &boss
			}
		} else if ki, ok := knownItems[itemName]; ok {
			item.ItemID = ki.id
			item.Source = ki.source
			item.SourceType = ki.sourceType
			item.SourceName = ki.sourceName
			item.IsTier = ki.isTier
			if ki.bossName != "" {
				b := ki.bossName
				item.BossName = &b
			}
		}
		seen[slotID] = true
		items = append(items, item)
	}

	for i, line := range lines {
		if i == 0 {
			continue // header
		}

		cols := strings.Split(line, "\t")
		if len(cols) < 2 {
			continue
		}

		slotRaw := strings.TrimSpace(cols[0])
		rawCell := strings.TrimSpace(cols[1])

		if slotRaw == "" || rawCell == "" {
			continue
		}

		slotKey := strings.ToLower(slotRaw)

		if isSkippedRow(slotKey) {
			continue
		}

		// Multi-item pair cells (Rings / Top Trinkets)
		if multiItemSlots[slotKey] {
			baseSlot, ok := icyVeinsSlotMap[slotKey]
			if !ok {
				continue
			}
			pairSlot := pairFallback[baseSlot]
			names := splitMultiItems(rawCell)
			if len(names) >= 1 && !seen[baseSlot] {
				if n := cleanItemName(names[0]); n != "" {
					addItem(baseSlot, n)
				}
			}
			if len(names) >= 2 && !seen[pairSlot] {
				if n := cleanItemName(names[1]); n != "" {
					addItem(pairSlot, n)
				}
			}
			continue
		}

		// Weapon cells with (1H) and (OH) markers
		if slotKey == "weapons" && strings.Contains(rawCell, "(1H)") {
			if !seen[16] {
				if n := extractMarkedWeapon(rawCell, "(1H)"); n != "" {
					addItem(16, n)
				}
			}
			if !seen[17] && strings.Contains(rawCell, "(OH)") {
				if n := extractMarkedWeapon(rawCell, "(OH)"); n != "" {
					addItem(17, n)
				}
			}
			continue
		}

		// Normal single-slot row
		slotID, ok := icyVeinsSlotMap[slotKey]
		if !ok {
			continue
		}

		// Pair fallback for unlabelled repeated slots
		if seen[slotID] {
			if fallback, hasFallback := pairFallback[slotID]; hasFallback && !seen[fallback] {
				slotID = fallback
			} else {
				continue
			}
		}

		itemName := cleanItemName(rawCell)
		if itemName == "" {
			continue
		}

		addItem(slotID, itemName)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SlotID < items[j].SlotID
	})

	return items
}

// splitMultiItems splits a multi-item cell into individual item names.
func splitMultiItems(cell string) []string {
	raw := strings.Split(cell, "  ")
	var result []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// extractMarkedWeapon finds the item name before a given marker like "(1H)" or "(OH)".
func extractMarkedWeapon(cell, marker string) string {
	idx := strings.Index(cell, marker)
	if idx < 0 {
		return ""
	}
	before := strings.TrimSpace(cell[:idx])
	parts := strings.Split(before, "  ")
	last := strings.TrimSpace(parts[len(parts)-1])
	return cleanItemName(last)
}

// isSkippedRow returns true for slot labels that are alternatives or noise rows.
func isSkippedRow(slotKey string) bool {
	skipPatterns := []string{
		"alternative",
		"alt.",
		"alt ",
		"(alternative)",
		"trinket alt",
		"ranking",
		"pack leader",
		"bis ->",
		"bis-",
	}
	for _, p := range skipPatterns {
		if strings.Contains(slotKey, p) {
			return true
		}
	}
	return false
}

// cleanItemName strips noise from item names.
func cleanItemName(name string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "your ") {
		return ""
	}
	trimmed := strings.TrimSpace(name)
	if strings.HasPrefix(strings.ToLower(trimmed), "bis -> ") {
		name = trimmed[7:]
	}
	if idx := strings.Index(name, " with "); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, ", or"); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, " ("); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "  "); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "\n"); idx > 0 {
		name = name[:idx]
	}
	return strings.TrimSpace(name)
}

// ============================================================
// Known items
// ============================================================

type knownItemDef struct {
	id         int
	sourceType string
	sourceName string
	source     string
	bossName   string
	isTier     bool
}

var knownItems = map[string]knownItemDef{

	// Blacksmithing
	"Spellbreaker's Bracers":      {id: 237834, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Spellbreaker's Warglaive":    {id: 237840, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Blood Knight's Impetus":      {id: 237847, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Mercy":          {id: 237837, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Chopper":        {id: 237850, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Plated Bracers": {id: 244584, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Aln'hara Lantern":            {id: 245769, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Spellbreaker's Ultimatum":    {id: 237841, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Aln'hara Cane":               {id: 245770, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Leatherworking
	"World Tender's Barkclasp":        {id: 244611, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"World Tender's Rootslippers":     {id: 244610, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Sneakers":     {id: 244569, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Deflectors":   {id: 244576, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Utility Belt": {id: 244573, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Handwraps":    {id: 244575, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Tailoring
	"Arcanoweave Bracers":      {id: 239660, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Arcanoweave Cloak":        {id: 239661, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Arcanoweave Cord":         {id: 239664, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Adherent's Silken Shroud": {id: 239656, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Martyr's Bindings":        {id: 239648, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Jewelcrafting
	"Platinum Star Band":          {id: 193708, sourceType: "DUNGEON", sourceName: "Algeth'ar Academy", source: "algetharacademy", bossName: "Vexamus"},
	"Loa Worshiper's Band":        {id: 251513, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Masterwork Sin'dorei Amulet": {id: 240950, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Masterwork Sin'dorei Band":   {id: 240949, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// March on Quel'Danas drops
	"Amulet of the Abyssal Hymn": {id: 250247, sourceType: "RAID", sourceName: "March on Quel'Danas", source: "marchonqueldanas", bossName: "Midnight Falls"},
	"Sin'dorei Band of Hope":     {id: 249919, sourceType: "RAID", sourceName: "March on Quel'Danas", source: "marchonqueldanas", bossName: "Child of Belo'ren"},

	// The Voidspire drops
	"Fallen King's Cuffs":           {id: 249304, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", bossName: "Fallen-King Salhadaar"},
	"Grimoire of the Eternal Light": {id: 249276, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", bossName: "Vorasius"},

	// Death Knight — Relentless Rider's Lament
	"Relentless Rider's Crown":       {id: 249970, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Dreadthorns": {id: 249968, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Cuirass":     {id: 249973, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Bonegrasps":  {id: 249971, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Legguards":   {id: 249969, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// Demon Hunter — Devouring Reaver's Sheathe
	"Devouring Reaver's Intake":          {id: 250033, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Exhaustplates":   {id: 250031, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Engine":          {id: 250036, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Devouring Reaver's Essence Grips":   {id: 250034, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Pistons":         {id: 250032, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Soul Flatteners": {id: 250035, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Druid — Sprouts of the Luminous Bloom
	"Branches of the Luminous Bloom":     {id: 250024, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Seedpods of the Luminous Bloom":     {id: 250022, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Trunk of the Luminous Bloom":        {id: 250027, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Arbortenders of the Luminous Bloom": {id: 250025, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Phloemwraps of the Luminous Bloom":  {id: 250023, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Leafdrape of the Luminous Bloom":    {id: 250019, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Evoker — Livery of the Black Talon
	"Hornhelm of the Black Talon":         {id: 249997, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Beacons of the Black Talon":          {id: 249995, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Frenzyward of the Black Talon":       {id: 250000, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Enforcer's Grips of the Black Talon": {id: 249998, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Greaves of the Black Talon":          {id: 249996, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Spelltreads of the Black Talon":      {id: 249999, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Hunter — Primal Sentry's Camouflage
	"Primal Sentry's Maw":         {id: 249988, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Primal Sentry's Scaleplate":  {id: 249991, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Primal Sentry's Talonguards": {id: 249989, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Primal Sentry's Legguards":   {id: 249987, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// Mage — Voidbreaker's Accordance
	"Voidbreaker's Veil":         {id: 250060, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Leyline Nexi": {id: 250058, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Robe":         {id: 250063, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Voidbreaker's Gloves":       {id: 250061, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Britches":     {id: 250059, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Sage Cord":    {id: 250057, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Voidbreaker's Treads":       {id: 250062, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Monk — Way of Ra-den's Chosen
	"Fearsome Visage of Ra-den's Chosen": {id: 250015, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Aurastones of Ra-den's Chosen":      {id: 250013, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Battle Garb of Ra-den's Chosen":     {id: 250018, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Thunderfists of Ra-den's Chosen":    {id: 250016, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Swiftsweepers of Ra-den's Chosen":   {id: 250014, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Storm Crashers of Ra-den's Chosen":  {id: 250017, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Strikeguards of Ra-den's Chosen":    {id: 250011, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Windwrap of Ra-den's Chosen":        {id: 250010, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Paladin — Luminant Verdict's Vestments
	"Luminant Verdict's Unwavering Gaze":  {id: 249961, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Providence Watch": {id: 249959, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Divine Warplate":  {id: 249964, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Luminant Verdict's Gauntlets":        {id: 249962, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Greaves":          {id: 249960, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// Priest — Blind Oath's Burden
	"Blind Oath's Winged Crest": {id: 250051, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Seraphguards": {id: 250049, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Raiment":      {id: 250054, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Blind Oath's Leggings":     {id: 250050, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Touch":        {id: 250052, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Wraps":        {id: 250047, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Rogue — Motley of the Grim Jest
	"Masquerade of the Grim Jest":       {id: 250006, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Venom Casks of the Grim Jest":      {id: 250004, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Fantastic Finery of the Grim Jest": {id: 250009, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Sleight of Hand of the Grim Jest":  {id: 250007, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blade Holsters of the Grim Jest":   {id: 250005, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Balancing Boots of the Grim Jest":  {id: 250008, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Shaman — Mantle of the Primal Core
	"Locus of the Primal Core":      {id: 249979, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Tempests of the Primal Core":   {id: 249977, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Embrace of the Primal Core":    {id: 249982, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Earthgrips of the Primal Core": {id: 249980, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Guardian of the Primal Core":   {id: 249974, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Ceinture of the Primal Core":   {id: 249976, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Leggings of the Primal Core":   {id: 249978, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// Warlock — Reign of the Abyssal Immolator
	"Abyssal Immolator's Smoldering Flames": {id: 250042, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Dreadrobe":         {id: 250045, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Abyssal Immolator's Grasps":            {id: 250043, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Pillars":           {id: 250041, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Blazing Core":      {id: 250039, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// API canonical name is "Scabrous Zombie Belt"; Icy Veins uses "Scabrous Zombie Leather Belt".
	// Alias at scraper boundary handles the Icy Veins variant.
	"Scabrous Zombie Belt": {id: 49810, sourceType: "DUNGEON", sourceName: "Pit of Saron", source: "pitofsaron", bossName: "Ick"},

	"Night Ender's Tusks":       {id: 249952, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Pauldrons":   {id: 249950, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Breastplate": {id: 249955, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Night Ender's Chausses":    {id: 249951, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Fists":       {id: 249953, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Girdle":      {id: 249949, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Night Ender's Greatboots":  {id: 249954, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
}

// ============================================================
// Item validator
// ============================================================

// fetchItemWithRetry calls fetchItem with exponential backoff for transient errors.
// On 429 responses, backoff floor is raised to 5s. On 404, returns immediately
// (not transient). Max 3 retries.
func (c *BlizzardClient) fetchItemWithRetry(itemID int) (*BlizzardItem, error) {
	maxRetries := 3
	backoff := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		item, err := c.fetchItem(itemID)
		if err == nil {
			return item, nil
		}

		errStr := err.Error()

		// 404 = item doesn't exist at this ID. Not transient — stop retrying.
		if strings.Contains(errStr, "not found:") {
			return nil, fmt.Errorf("404 not found")
		}

		// 429 = rate limited. Raise backoff floor to 5s and retry.
		if strings.Contains(errStr, "HTTP 429") && attempt < maxRetries {
			if backoff < 5*time.Second {
				backoff = 5 * time.Second
			}
			continue
		}

		// 5xx or network errors — transient, keep retrying.
		if attempt < maxRetries {
			continue
		}

		return nil, fmt.Errorf("API error after %d retries: %s", maxRetries, errStr)
	}

	return nil, fmt.Errorf("unreachable")
}

// normalizeItemName prepares an item name for comparison. Lowercases, trims
// whitespace, and normalises unicode punctuation (smart quotes etc).
func normalizeItemName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "\u2019", "'") // right single quote → apostrophe
	s = strings.ReplaceAll(s, "\u2018", "'") // left single quote → apostrophe
	s = strings.ReplaceAll(s, "\u201c", "\"")
	s = strings.ReplaceAll(s, "\u201d", "\"")
	return s
}

type validationResult struct {
	Name    string `json:"name"`
	ID      int    `json:"id"`
	Status  string `json:"status"`  // "PASS", "FAIL", "SKIP"
	APIName string `json:"apiName"` // name returned by API (for FAIL)
	Detail  string `json:"detail"`  // error detail (for SKIP)
	IsTier  bool   `json:"isTier"`
}

const validationResultsPath = "data/validation_results.json"

func validateItems(strict bool) error {
	fmt.Println("=== GearPath knownItems Validator ===")
	fmt.Println()

	fmt.Print("Authenticating with Blizzard API... ")
	blizzard, err := newBlizzardClient()
	if err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}
	fmt.Println("OK")
	fmt.Println()

	// Sort keys for deterministic output.
	names := make([]string, 0, len(knownItems))
	for name := range knownItems {
		names = append(names, name)
	}
	sort.Strings(names)

	// Duplicate ID check — run before API validation.
	idToNames := map[int][]string{}
	for name, ki := range knownItems {
		idToNames[ki.id] = append(idToNames[ki.id], name)
	}
	dupeGroups := 0
	for id, owners := range idToNames {
		if len(owners) > 1 {
			sort.Strings(owners)
			if dupeGroups == 0 {
				fmt.Println("=== Duplicate IDs ===")
			}
			fmt.Printf("  id=%-8d shared by: %s\n", id, strings.Join(owners, ", "))
			dupeGroups++
		}
	}
	if dupeGroups > 0 {
		fmt.Printf("  %d group(s) of duplicate IDs found — at least one entry per group is wrong.\n\n", dupeGroups)
	} else {
		fmt.Printf("No duplicate IDs found.\n\n")
	}

	// Alias integrity check — every alias value must exist in knownItems.
	badAliases := 0
	for from, to := range nameAliases {
		if _, ok := knownItems[to]; !ok {
			if badAliases == 0 {
				fmt.Println("=== Broken Aliases ===")
			}
			fmt.Printf("  %q → %q (target not in knownItems)\n", from, to)
			badAliases++
		}
	}
	if badAliases > 0 {
		fmt.Printf("  %d broken alias(es) — scraper will silently drop these items.\n\n", badAliases)
	}

	var results []validationResult
	passed, failed, skipped := 0, 0, 0

	fmt.Printf("Validating %d items against Blizzard API...\n\n", len(names))

	for i, name := range names {
		ki := knownItems[name]
		fmt.Printf("  [%3d/%d] %-50s id=%-8d ", i+1, len(names), name, ki.id)

		item, err := blizzard.fetchItemWithRetry(ki.id)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "404") {
				fmt.Printf("SKIP (404 — item not found)\n")
				results = append(results, validationResult{
					Name: name, ID: ki.id, Status: "SKIP",
					Detail: "404 not found", IsTier: ki.isTier,
				})
			} else {
				fmt.Printf("SKIP (API error: %s)\n", errStr)
				results = append(results, validationResult{
					Name: name, ID: ki.id, Status: "SKIP",
					Detail: errStr, IsTier: ki.isTier,
				})
			}
			skipped++
			continue
		}

		if normalizeItemName(item.Name) == normalizeItemName(name) {
			fmt.Printf("PASS\n")
			results = append(results, validationResult{
				Name: name, ID: ki.id, Status: "PASS",
				APIName: item.Name, IsTier: ki.isTier,
			})
			passed++
		} else {
			fmt.Printf("FAIL — API says: %q\n", item.Name)
			results = append(results, validationResult{
				Name: name, ID: ki.id, Status: "FAIL",
				APIName: item.Name, IsTier: ki.isTier,
			})
			failed++
		}

		// Rate limit: 30ms between calls (matches existing sourceMap pattern).
		time.Sleep(30 * time.Millisecond)
	}

	// Print summary.
	fmt.Println()
	fmt.Println("=== Validation Summary ===")
	fmt.Printf("  Total:   %d\n", len(names))
	fmt.Printf("  PASS:    %d\n", passed)
	fmt.Printf("  FAIL:    %d\n", failed)
	fmt.Printf("  SKIP:    %d\n", skipped)
	fmt.Println()

	if failed > 0 {
		fmt.Println("=== FAILURES (name mismatch) ===")
		for _, r := range results {
			if r.Status == "FAIL" {
				tierTag := ""
				if r.IsTier {
					tierTag = " [TIER]"
				}
				fmt.Printf("  %-50s id=%-8d → API name: %q%s\n", r.Name, r.ID, r.APIName, tierTag)
			}
		}
		fmt.Println()
	}

	if skipped > 0 {
		fmt.Println("=== SKIPPED (API errors / 404) ===")
		for _, r := range results {
			if r.Status == "SKIP" {
				tierTag := ""
				if r.IsTier {
					tierTag = " [TIER]"
				}
				fmt.Printf("  %-50s id=%-8d → %s%s\n", r.Name, r.ID, r.Detail, tierTag)
			}
		}
		fmt.Println()
	}

	// Dump results to JSON for use by resolve-items.
	if data, err := json.MarshalIndent(results, "", "  "); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal validation results: %v\n", err)
	} else {
		if err := os.MkdirAll(filepath.Dir(validationResultsPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory for %s: %v\n", validationResultsPath, err)
		} else if err := os.WriteFile(validationResultsPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", validationResultsPath, err)
		} else {
			fmt.Printf("Results written to %s\n\n", validationResultsPath)
		}
	}

	if failed > 0 {
		fmt.Printf("VALIDATION FAILED: %d item(s) have mismatched names.\n", failed)
		fmt.Println("The knownItems map has incorrect ID-to-name associations.")
		fmt.Println("Fix the entries above before regenerating data.")
		return fmt.Errorf("%d validation failure(s)", failed)
	}

	if strict && skipped > 0 {
		fmt.Printf("STRICT MODE: %d item(s) could not be verified.\n", skipped)
		return fmt.Errorf("%d unverifiable item(s) in strict mode", skipped)
	}

	if skipped > 0 {
		fmt.Printf("All verifiable items passed. %d item(s) skipped (review above).\n", skipped)
	} else {
		fmt.Println("All items validated successfully. ✓")
	}

	return nil
}

// ============================================================
// Item resolver (scan-based)
// ============================================================

// maxScanRange caps how wide a neighborhood scan can be. If PASSed siblings
// span a wider range than this, the item is skipped (resolve manually).
const maxScanRange = 30

// extractNameFamily returns the shared family name for grouping siblings.
//
// "of" suffix checked first: "Strikeguards of Ra-den's Chosen" → "of Ra-den's Chosen"
// This must precede the possessive check because set names like "Ra-den's" contain
// an apostrophe that would incorrectly match the possessive path.
//
// Possessive prefix: "Relentless Rider's Crown" → "Relentless Rider's"
// Neither: returns "" (fall through to sourceType grouping).
func extractNameFamily(name string) string {
	if idx := strings.Index(name, " of the "); idx >= 0 {
		return name[idx+1:]
	}
	if idx := strings.Index(name, " of "); idx >= 0 {
		return name[idx+1:]
	}
	if idx := strings.LastIndex(name, "'s "); idx >= 0 {
		return name[:idx+2]
	}
	return ""
}

func loadValidationResults() ([]validationResult, error) {
	raw, err := os.ReadFile(validationResultsPath)
	if err != nil {
		return nil, fmt.Errorf("run 'go run . validate-items' first: %w", err)
	}

	info, err := os.Stat(validationResultsPath)
	if err == nil && time.Since(info.ModTime()) > 10*time.Minute {
		fmt.Fprintf(os.Stderr, "Warning: %s is older than 10 minutes — consider re-running validate-items\n\n", validationResultsPath)
	}

	var results []validationResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", validationResultsPath, err)
	}
	return results, nil
}

type sibling struct {
	name string
	id   int
}

func resolveItemsScan(names []string) error {
	fmt.Println("=== GearPath Item Resolver (scan) ===")
	fmt.Println()

	results, err := loadValidationResults()
	if err != nil {
		return err
	}

	// Build set of PASSed items: name → id.
	passed := map[string]int{}
	for _, r := range results {
		if r.Status == "PASS" {
			passed[r.Name] = r.ID
		}
	}
	fmt.Printf("Loaded %d PASSed items from %s\n\n", len(passed), validationResultsPath)

	fmt.Print("Authenticating with Blizzard API... ")
	blizzard, err := newBlizzardClient()
	if err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}
	fmt.Println("OK")
	fmt.Println()

	resolved, ambiguous, notFound, skippedCount := 0, 0, 0, 0

	for i, name := range names {
		fmt.Printf("[%2d/%d] %s\n", i+1, len(names), name)

		// Single pass: find PASSed siblings by name family.
		family := extractNameFamily(name)
		var siblings []sibling

		if family != "" {
			for pName, pID := range passed {
				if extractNameFamily(pName) == family {
					siblings = append(siblings, sibling{name: pName, id: pID})
				}
			}
		}

		// Fallback: same sourceType+sourceName from knownItems.
		if len(siblings) == 0 {
			if ki, ok := knownItems[name]; ok {
				for pName, pID := range passed {
					if pki, ok2 := knownItems[pName]; ok2 {
						if pki.sourceType == ki.sourceType && pki.sourceName == ki.sourceName {
							siblings = append(siblings, sibling{name: pName, id: pID})
						}
					}
				}
			}
		}

		if len(siblings) == 0 {
			fmt.Printf("       SKIP (no PASSed siblings to derive scan range)\n")
			skippedCount++
			continue
		}

		// Derive scan range from sibling IDs.
		sort.Slice(siblings, func(a, b int) bool { return siblings[a].id < siblings[b].id })
		lo := siblings[0].id - 5
		hi := siblings[len(siblings)-1].id + 5

		if hi-lo > maxScanRange {
			fmt.Printf("       SKIP (sibling range %d–%d exceeds %d-ID cap)\n", lo, hi, maxScanRange)
			skippedCount++
			continue
		}

		// Print sibling summary.
		sibLabels := make([]string, len(siblings))
		for j, s := range siblings {
			sibLabels[j] = fmt.Sprintf("%s(%d)", s.name, s.id)
		}
		fmt.Printf("       siblings: %s\n", strings.Join(sibLabels, ", "))
		fmt.Printf("       scan range: %d–%d\n", lo, hi)

		// Scan the range.
		// Scan the range. No ilvl filter needed — the sibling-derived neighborhood
		// already constrains to Midnight-era IDs. The individual item API returns
		// base ilvl (e.g. 197 for tier), not the difficulty-scaled display ilvl.
		want := normalizeItemName(name)
		type scanMatch struct {
			id    int
			name  string
			level int
		}
		var matches []scanMatch
		for id := lo; id <= hi; id++ {
			item, err := blizzard.fetchItemWithRetry(id)
			if err != nil {
				continue
			}
			if normalizeItemName(item.Name) == want {
				matches = append(matches, scanMatch{id: item.ID, name: item.Name, level: item.Level})
			}
			time.Sleep(30 * time.Millisecond)
		}

		if len(matches) == 0 {
			fmt.Printf("       NOT_FOUND (scanned %d–%d, no exact match)\n", lo, hi)
			notFound++
		} else if len(matches) == 1 {
			m := matches[0]
			fmt.Printf("       RESOLVED  id=%-8d ilvl=%-4d name=%q\n", m.id, m.level, m.name)
			resolved++
		} else {
			fmt.Printf("       AMBIGUOUS (%d matches — needs human review)\n", len(matches))
			for _, m := range matches {
				fmt.Printf("         id=%-8d ilvl=%-4d name=%q\n", m.id, m.level, m.name)
			}
			ambiguous++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("  RESOLVED:  %d\n", resolved)
	fmt.Printf("  AMBIGUOUS: %d\n", ambiguous)
	fmt.Printf("  NOT_FOUND: %d\n", notFound)
	fmt.Printf("  SKIPPED:   %d\n", skippedCount)

	return nil
}

// ============================================================
// Unresolved item tracker
// ============================================================

type unresolvedItem struct {
	slot string
	name string
}

func findUnresolvedItems(updatedDate, target string) []unresolvedItem {
	slotNames := map[int]string{
		1: "Head", 2: "Neck", 3: "Shoulders", 5: "Chest", 6: "Waist",
		7: "Legs", 8: "Feet", 9: "Wrists", 10: "Hands", 11: "Finger 1",
		12: "Finger 2", 13: "Trinket 1", 14: "Trinket 2", 15: "Back",
		16: "Main Hand", 17: "Off Hand",
	}
	seen := map[string]bool{}
	var result []unresolvedItem

	files, _ := filepath.Glob("data/*.json")
	for _, f := range files {
		base := filepath.Base(f)
		if base == filepath.Base(sourceMapCachePath) {
			continue
		}
		if base == filepath.Base(validationResultsPath) {
			continue
		}
		// Skip guide files — they have a different format and don't contain BiS items.
		if strings.HasPrefix(base, "guide_") {
			continue
		}
		spec, err := loadSpec(f)
		if err != nil {
			continue
		}
		// Target can be "CLASS_Spec" or "CLASS_Spec_Hero" — match either.
		specKey := spec.Class + "_" + spec.Spec
		fullKey := specKey + "_" + spec.HeroKey
		if target != "" && target != specKey && target != fullKey {
			continue
		}
		if updatedDate != "" && spec.Updated != updatedDate {
			continue
		}
		for _, item := range spec.Items {
			if item.ItemID == 0 && !seen[item.ItemName] {
				seen[item.ItemName] = true
				result = append(result, unresolvedItem{
					slot: slotNames[item.SlotID],
					name: item.ItemName,
				})
			}
		}
	}
	return result
}

// ============================================================
// List instances (debug)
// ============================================================

func listInstances() error {
	clientID := os.Getenv("BLIZZARD_CLIENT_ID")
	clientSecret := os.Getenv("BLIZZARD_CLIENT_SECRET")
	c := &BlizzardClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		sourceMap:  map[int]ItemSource{},
		nameMap:    map[string]int{},
	}
	if err := c.authenticate(clientID, clientSecret); err != nil {
		return err
	}
	var index JournalInstanceIndex
	if err := c.get(
		"https://us.api.blizzard.com/data/wow/journal-instance/index?namespace=static-us&locale=en_US",
		&index,
	); err != nil {
		return err
	}
	for _, inst := range index.Instances {
		fmt.Printf("%d\t%s\n", inst.ID, inst.Name)
	}
	return nil
}

// ============================================================
// Spec selection
// ============================================================

type specSelection struct {
	Class   string
	Spec    string
	HeroKey string
}

func selectSpec(reader *bufio.Reader) *specSelection {
	fmt.Println("\nSelect a spec to edit:")
	fmt.Println()

	// Build a flat list of editable targets (spec × hero, or spec × "any").
	// This mirrors expandScrapeTargets so the editor can reach split specs.
	targets := expandScrapeTargets()
	for i, t := range targets {
		filename := buildDataFilename(t.Class, t.Spec, t.HeroKey)
		status := "empty"
		if spec, err := loadSpec(filename); err == nil && len(spec.Items) > 0 {
			status = fmt.Sprintf("%d/16 slots", len(spec.Items))
		}
		label := fmt.Sprintf("%s %s", t.Class, t.Spec)
		if t.HeroKey != "any" {
			label += " (" + t.HeroKey + ")"
		}
		fmt.Printf("  %2d. %-45s [%s]\n", i+1, label, status)
	}

	fmt.Println()
	fmt.Print("Enter number (q to quit): ")
	input := readLine(reader)

	if strings.ToLower(input) == "q" {
		return nil
	}

	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(targets) {
		fmt.Println("Invalid selection.")
		return nil
	}

	t := targets[n-1]
	return &specSelection{Class: t.Class, Spec: t.Spec, HeroKey: t.HeroKey}
}

// ============================================================
// Spec editing
// ============================================================

func editSpec(reader *bufio.Reader, blizzard *BlizzardClient, class, spec, heroKey string) {
	filename := buildDataFilename(class, spec, heroKey)

	specData, err := loadSpec(filename)
	if err != nil {
		specData = &SpecData{
			Class:   class,
			Spec:    spec,
			HeroKey: heroKey,
			Season:  "Midnight S1",
			Updated: time.Now().Format("2006-01-02"),
			Items:   []ItemData{},
		}
	}
	// Guard against old data files that pre-date the heroKey field.
	if specData.HeroKey == "" {
		specData.HeroKey = heroKey
	}

	for {
		heroLabel := ""
		if specData.HeroKey != "any" {
			heroLabel = " (" + specData.HeroKey + ")"
		}
		fmt.Printf("\n═══ %s %s%s ═══\n\n", specData.Class, specData.Spec, heroLabel)

		for i, slot := range slots {
			item := findItem(specData, slot.ID)
			if item != nil {
				tier := ""
				if item.IsTier {
					tier = " [TIER]"
				}
				fmt.Printf("  %2d. [%-10s] %-42s (ID: %d, ilvl: %d)%s\n",
					i+1, slot.Name, item.ItemName, item.ItemID, item.Ilvl, tier)
			} else {
				fmt.Printf("  %2d. [%-10s] —\n", i+1, slot.Name)
			}
		}

		fmt.Println()
		fmt.Print("Action — slot number to edit, c=clear, s=save, q=back: ")
		input := readLine(reader)

		switch strings.ToLower(input) {
		case "q":
			return
		case "s":
			specData.Updated = time.Now().Format("2006-01-02")
			if err := saveSpec(filename, specData); err != nil {
				fmt.Printf("Error saving: %v\n", err)
			} else {
				fmt.Println("Saved.")
			}
			return
		case "c":
			fmt.Print("Clear which slot number? ")
			cn := readLine(reader)
			n, err := strconv.Atoi(cn)
			if err != nil || n < 1 || n > len(slots) {
				fmt.Println("Invalid.")
				continue
			}
			clearItem(specData, slots[n-1].ID)
			fmt.Printf("Cleared: %s\n", slots[n-1].Name)
		default:
			n, err := strconv.Atoi(input)
			if err != nil || n < 1 || n > len(slots) {
				fmt.Println("Invalid.")
				continue
			}
			editSlot(reader, blizzard, specData, slots[n-1])
		}
	}
}

// ============================================================
// Slot editing
// ============================================================

func editSlot(reader *bufio.Reader, blizzard *BlizzardClient, spec *SpecData, slot struct {
	ID   int
	Name string
}) {
	existing := findItem(spec, slot.ID)
	fmt.Printf("\nEditing: %s", slot.Name)
	if existing != nil {
		fmt.Printf(" (current: %s, ID %d)", existing.ItemName, existing.ItemID)
	}
	fmt.Println()

	fmt.Print("Item ID (Enter to cancel): ")
	input := readLine(reader)
	if input == "" {
		return
	}

	itemID, err := strconv.Atoi(input)
	if err != nil || itemID <= 0 {
		fmt.Println("Invalid item ID.")
		return
	}

	fmt.Printf("Looking up item %d... ", itemID)
	blizzItem, err := blizzard.fetchItem(itemID)
	if err != nil {
		fmt.Printf("FAILED (%v)\n", err)
		return
	}
	fmt.Printf("found: %s (ilvl %d)\n", blizzItem.Name, blizzItem.Level)

	item := ItemData{
		SlotID:   slot.ID,
		ItemID:   itemID,
		ItemName: blizzItem.Name,
		Ilvl:     blizzItem.Level,
		Priority: priorityForSlot(slot.ID),
	}

	if src, ok := blizzard.sourceMap[itemID]; ok {
		item.Source = src.Source
		item.SourceType = src.SourceType
		item.SourceName = src.SourceName
		boss := src.BossName
		item.BossName = &boss
		fmt.Printf("Source:  %s — %s (%s)\n", src.SourceName, src.BossName, src.SourceType)
	} else if blizzItem.PreviewItem.Source.Type == "CREATED_BY_SPELL" {
		item.Source = "Crafted"
		item.SourceType = "CRAFTED"
		item.SourceName = "Crafted"
		item.BossName = nil
		fmt.Println("Source:  Crafted item (profession)")
	} else {
		item.BossName = nil
		fmt.Println("Source:  Not found in Season 1 journal — what type of item is this?")
		fmt.Println("  1. Crafted (profession)")
		fmt.Println("  2. World Drop")
		fmt.Println("  3. Great Vault only")
		fmt.Print("Choice (default=1): ")
		choice := readLine(reader)
		switch choice {
		case "2":
			item.Source = "WorldDrop"
			item.SourceType = "WORLD"
			item.SourceName = "World Drop"
		case "3":
			item.Source = "GreatVault"
			item.SourceType = "VAULT"
			item.SourceName = "Great Vault"
		default:
			item.Source = "Crafted"
			item.SourceType = "CRAFTED"
			item.SourceName = "Crafted"
		}
		fmt.Printf("Source set to: %s\n", item.SourceName)
	}

	fmt.Print("Is this a tier set piece? (y/n): ")
	item.IsTier = strings.ToLower(readLine(reader)) == "y"
	if item.IsTier && item.SourceType != "RAID" {
		fmt.Println("Tier pieces drop from raid bosses and the Great Vault.")
		fmt.Print("Which raid? (1=The Voidspire, 2=The Dreamrift, 3=March on Quel'Danas): ")
		choice := readLine(reader)
		raids := map[string]struct{ name, source string }{
			"1": {"The Voidspire", "thevoidspire"},
			"2": {"The Dreamrift", "thedreamrift"},
			"3": {"March on Quel'Danas", "marchonqueldanas"},
		}
		if r, ok := raids[choice]; ok {
			item.Source = r.source
			item.SourceType = "RAID"
			item.SourceName = r.name
			item.BossName = nil
			fmt.Printf("Source set to: %s [RAID]\n", r.name)
		}
	}

	fmt.Printf("\n  %-10s %s (ID: %d, ilvl: %d)\n", slot.Name+":", item.ItemName, item.ItemID, item.Ilvl)
	fmt.Printf("  Source:    %s", item.SourceName)
	if item.BossName != nil {
		fmt.Printf(" — %s", *item.BossName)
	}
	fmt.Printf(" [%s]\n", item.SourceType)
	if item.IsTier {
		fmt.Println("  Tier:      Yes")
	}

	fmt.Print("\nConfirm? (y/n): ")
	if strings.ToLower(readLine(reader)) != "y" {
		fmt.Println("Cancelled.")
		return
	}

	setItem(spec, item)
	fmt.Printf("✓ %s updated.\n", slot.Name)
}

// ============================================================
// Helpers
// ============================================================

func priorityForSlot(slotID int) int {
	switch slotID {
	case 16, 17:
		return 10
	case 13, 14:
		return 10
	case 1, 3, 5, 10, 7:
		return 9
	case 2, 11, 12:
		return 7
	default:
		return 6
	}
}

func findItem(spec *SpecData, slotID int) *ItemData {
	for i := range spec.Items {
		if spec.Items[i].SlotID == slotID {
			return &spec.Items[i]
		}
	}
	return nil
}

func setItem(spec *SpecData, item ItemData) {
	for i := range spec.Items {
		if spec.Items[i].SlotID == item.SlotID {
			spec.Items[i] = item
			return
		}
	}
	spec.Items = append(spec.Items, item)
	sort.Slice(spec.Items, func(i, j int) bool {
		return spec.Items[i].SlotID < spec.Items[j].SlotID
	})
}

func clearItem(spec *SpecData, slotID int) {
	var filtered []ItemData
	for _, item := range spec.Items {
		if item.SlotID != slotID {
			filtered = append(filtered, item)
		}
	}
	spec.Items = filtered
}

func loadSpec(filename string) (*SpecData, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var spec SpecData
	if err := json.NewDecoder(f).Decode(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func saveSpec(filename string, spec *SpecData) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// ============================================================
// Generate
// ============================================================

func generate() error {
	jsonFiles, err := filepath.Glob("data/*.json")
	if err != nil {
		return fmt.Errorf("could not read data directory: %w", err)
	}
	if len(jsonFiles) == 0 {
		return fmt.Errorf("no JSON files found in data/")
	}

	sort.Strings(jsonFiles)

	// Load all BiS specs and group by {class, spec} so we can emit one Lua
	// block per spec with nested hero keys.
	type specKey struct{ Class, Spec string }
	grouped := map[specKey]map[string]*SpecData{}

	for _, file := range jsonFiles {
		base := filepath.Base(file)
		if base == filepath.Base(sourceMapCachePath) {
			continue
		}
		if base == filepath.Base(validationResultsPath) {
			continue
		}
		if strings.HasPrefix(base, "guide_") {
			continue
		}
		spec, err := loadSpec(file)
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
		if len(spec.Items) == 0 {
			fmt.Printf("  Skipping (empty): %s %s [%s]\n", spec.Class, spec.Spec, spec.HeroKey)
			continue
		}
		heroKey := spec.HeroKey
		if heroKey == "" {
			heroKey = "any" // backward compat with data files predating heroKey
		}
		sort.Slice(spec.Items, func(i, j int) bool {
			return spec.Items[i].SlotID < spec.Items[j].SlotID
		})
		k := specKey{spec.Class, spec.Spec}
		if grouped[k] == nil {
			grouped[k] = map[string]*SpecData{}
		}
		grouped[k][heroKey] = spec
		fmt.Printf("  Loaded: %s %s [%s] (%d items)\n", spec.Class, spec.Spec, heroKey, len(spec.Items))
	}

	if len(grouped) == 0 {
		return fmt.Errorf("no specs with items found")
	}

	// Flatten into a stable-ordered slice of SpecGroups for the template.
	var keys []specKey
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Class != keys[j].Class {
			return keys[i].Class < keys[j].Class
		}
		return keys[i].Spec < keys[j].Spec
	})

	var groups []SpecGroup
	for _, k := range keys {
		heroMap := grouped[k]
		var heroKeys []string
		for h := range heroMap {
			heroKeys = append(heroKeys, h)
		}
		sort.Strings(heroKeys)

		g := SpecGroup{
			Class:  k.Class,
			Spec:   k.Spec,
			Season: "Midnight S1",
		}
		for _, h := range heroKeys {
			s := heroMap[h]
			g.Heroes = append(g.Heroes, HeroGroup{
				HeroKey: h,
				Updated: s.Updated,
				Items:   s.Items,
			})
		}
		groups = append(groups, g)
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

	if err := tmpl.Execute(outFile, TemplateData{
		Generated: time.Now().Format("2006-01-02 15:04:05"),
		Groups:    groups,
	}); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	totalLists := 0
	for _, g := range groups {
		totalLists += len(g.Heroes)
	}
	fmt.Printf("\nGenerated: %s (%d specs, %d hero lists)\n", outputPath, len(groups), totalLists)
	return nil
}

// ============================================================
// Init-specs
// ============================================================

func initSpecs() error {
	today := time.Now().Format("2006-01-02")
	created, skipped := 0, 0

	for _, t := range expandScrapeTargets() {
		filename := buildDataFilename(t.Class, t.Spec, t.HeroKey)
		if _, err := os.Stat(filename); err == nil {
			skipped++
			continue
		}
		skeleton := SpecData{
			Class:   t.Class,
			Spec:    t.Spec,
			HeroKey: t.HeroKey,
			Season:  "Midnight S1",
			Updated: today,
			Items:   []ItemData{},
		}
		data, _ := json.MarshalIndent(skeleton, "", "  ")
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}
		fmt.Printf("  Created: %s\n", filename)
		created++
	}

	fmt.Printf("\nDone. Created: %d, Skipped: %d\n", created, skipped)
	return nil
}

// ============================================================
// Guide scraper — stat priority + gems/enchants/consumables
// (unchanged in this PR; hero-talent support for guides is deferred
// to a follow-up branch per the feat/hero-talents-guides plan)
// ============================================================

type GuideData struct {
	Class       string
	Spec        string
	Updated     string
	Stats       StatPriority
	Gems        []GemEntry
	Enchants    []EnchantEntry
	Consumables ConsumableSet
}

type StatPriority struct {
	Format  string
	Ordered []string
	Note    string
}

type GemEntry struct {
	Name string
	Note string
}

type EnchantEntry struct {
	Slot    string
	Enchant string
}

type ConsumableSet struct {
	Flask     string
	Food      string
	Potion    string
	WeaponOil string
	AugRune   string
}

type GuideEntry struct {
	HeroKey     string
	Updated     string
	Stats       StatPriority
	Gems        []GemEntry
	Enchants    []EnchantEntry
	Consumables ConsumableSet
}

type GuideGroup struct {
	Class   string
	Spec    string
	Entries []GuideEntry
}

type GuideTemplateData struct {
	Generated string
	Groups    []GuideGroup
}

const (
	statSuffix = "-stat-priority"
	gearSuffix = "-gems-enchants-consumables"
)

func scrapeGuideAll(target string) error {
	fmt.Println("Starting headless browser for guide scraping...")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	fmt.Println("Browser ready.")
	fmt.Println()

	today := time.Now().Format("2006-01-02")
	success, failed, skipped := 0, 0, 0

	for _, s := range allSpecs {
		key := s.Class + "_" + s.Spec
		if target != "" && key != target {
			continue
		}

		slug, ok := icyVeinsSlugs[key]
		if !ok {
			fmt.Printf("  No slug for %s — skipping\n", key)
			skipped++
			continue
		}

		fmt.Printf("Scraping guide %-14s %-14s ... ", s.Class, s.Spec)

		heroSpec, isHeroSplit := heroGuideSplits[key]
		if isHeroSplit {
			guides, err := scrapeGuideSpecHero(browserCtx, slug, s.Class, s.Spec, today, heroSpec)
			if err != nil {
				fmt.Printf("FAILED (%v)\n", err)
				failed++
				continue
			}
			writeErr := false
			for heroKey, guide := range guides {
				filename := fmt.Sprintf("data/guide_%s_%s_%s.json", s.Class, s.Spec, heroKey)
				data, _ := json.MarshalIndent(guide, "", "  ")
				if err := os.WriteFile(filename, data, 0644); err != nil {
					fmt.Printf("FAILED (write %s: %v)\n", heroKey, err)
					writeErr = true
					break
				}
			}
			if writeErr {
				failed++
				continue
			}
			fmt.Printf("OK (%d hero keys)\n", len(guides))
		} else {
			guide, err := scrapeGuideSpec(browserCtx, slug, s.Class, s.Spec, today)
			if err != nil {
				fmt.Printf("FAILED (%v)\n", err)
				failed++
				continue
			}
			filename := fmt.Sprintf("data/guide_%s_%s_any.json", s.Class, s.Spec)
			data, _ := json.MarshalIndent(guide, "", "  ")
			if err := os.WriteFile(filename, data, 0644); err != nil {
				fmt.Printf("FAILED (write: %v)\n", err)
				failed++
				continue
			}
			fmt.Printf("OK\n")
		}

		success++
		time.Sleep(1200 * time.Millisecond)
	}

	fmt.Printf("\nDone. OK: %d, Failed: %d, Skipped: %d\n", success, failed, skipped)

	if failed == 0 {
		fmt.Println("Run 'go run . generate-guide' to produce GearPath_Stats.lua")
	}
	return nil
}

func scrapeGuideSpec(browserCtx context.Context, slug, class, spec, today string) (*GuideData, error) {
	guide := &GuideData{
		Class:   class,
		Spec:    spec,
		Updated: today,
	}

	statURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s%s", slug, statSuffix)
	var statText string
	pageCtx, pageCancel := context.WithTimeout(browserCtx, 30*time.Second)
	err := chromedp.Run(pageCtx,
		chromedp.Navigate(statURL),
		chromedp.WaitVisible(".page_content", chromedp.ByQuery),
		chromedp.Sleep(400*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.page_content').innerText`, &statText),
	)
	pageCancel()
	if err != nil {
		return nil, fmt.Errorf("stat page: %w", err)
	}
	guide.Stats = parseStatPriority(statText)

	consumURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s%s", slug, gearSuffix)
	var consumText string
	pageCtx2, pageCancel2 := context.WithTimeout(browserCtx, 30*time.Second)
	err = chromedp.Run(pageCtx2,
		chromedp.Navigate(consumURL),
		chromedp.WaitVisible(".page_content", chromedp.ByQuery),
		chromedp.Sleep(400*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.page_content').innerText`, &consumText),
	)
	pageCancel2()
	if err != nil {
		return nil, fmt.Errorf("consumables page: %w", err)
	}

	guide.Gems = parseGems(consumText)
	guide.Enchants = parseEnchants(consumText)
	guide.Consumables = parseConsumables(consumText)

	warnGuideGaps(consumText, guide, class, spec)

	return guide, nil
}

// scrapeGuideSpecHero scrapes guide data for a spec with per-hero-talent
// content. Stats and consumables are handled independently:
//
// Stats: if tabs have StatSelector, clicks each tab and parses per hero.
// If StatSelector is empty, reads once and duplicates to all hero keys.
//
// Consumables: if ConsumSubTabsEnabled, clicks per-hero + per-content-type
// sub-tabs. Otherwise reads once and shares across hero keys.
//
// Returns a map of heroKey → *GuideData. If FallbackSelector is set, also
// includes an "any" entry from the General stat priority tab.
func scrapeGuideSpecHero(browserCtx context.Context, slug, class, spec, today string, heroSpec heroGuideSpec) (map[string]*GuideData, error) {
	tag := class + "_" + spec

	clickTab := func(ctx context.Context, selector string) error {
		var ok bool
		err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`(function(){ var el = document.querySelector('%s'); if(!el) return false; el.click(); return true; })()`, selector), &ok),
			chromedp.Sleep(500*time.Millisecond),
		)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("selector %q not found in DOM", selector)
		}
		return nil
	}

	readPageText := func(ctx context.Context) (string, error) {
		var text string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelector('.page_content').innerText`, &text),
		)
		return text, err
	}

	// =================================================================
	// STATS PHASE
	// =================================================================
	statURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s%s", slug, statSuffix)
	statCtx, statCancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer statCancel()

	err := chromedp.Run(statCtx,
		chromedp.Navigate(statURL),
		chromedp.WaitVisible(".page_content", chromedp.ByQuery),
		chromedp.Sleep(400*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("stat page navigate: %w", err)
	}

	heroStats := make(map[string]StatPriority)

	// Optional fallback: click the "General" tab and store as _any.
	if heroSpec.FallbackSelector != "" {
		if err := clickTab(statCtx, heroSpec.FallbackSelector); err != nil {
			fmt.Printf("WARN [%s] fallback stat tab click failed (%s): %v\n", tag, heroSpec.FallbackSelector, err)
		} else if text, err := readPageText(statCtx); err == nil {
			heroStats["any"] = parseStatPriority(text)
		}
	}

	// Check if any tab needs stat-page clicking.
	needsStatClicks := false
	for _, tab := range heroSpec.Tabs {
		if tab.StatSelector != "" {
			needsStatClicks = true
			break
		}
	}

	if needsStatClicks {
		for _, tab := range heroSpec.Tabs {
			if tab.StatSelector == "" {
				continue
			}
			if err := clickTab(statCtx, tab.StatSelector); err != nil {
				fmt.Printf("WARN [%s] stat tab click failed for %s (%s): %v\n", tag, tab.Key, tab.StatSelector, err)
				continue
			}
			text, err := readPageText(statCtx)
			if err != nil {
				fmt.Printf("WARN [%s] stat text read failed for %s: %v\n", tag, tab.Key, err)
				continue
			}
			heroStats[tab.Key] = parseStatPriority(text)
		}
	} else {
		// No stat tabs: single read, parse once, store for all hero keys.
		text, err := readPageText(statCtx)
		if err != nil {
			return nil, fmt.Errorf("stat page read: %w", err)
		}
		parsed := parseStatPriority(text)
		for _, tab := range heroSpec.Tabs {
			heroStats[tab.Key] = parsed
		}
	}

	if len(heroStats) == 0 {
		return nil, fmt.Errorf("no hero talent stats captured")
	}

	// =================================================================
	// CONSUMABLES PHASE
	// =================================================================
	consumURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s%s", slug, gearSuffix)
	consumCtx, consumCancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer consumCancel()

	err = chromedp.Run(consumCtx,
		chromedp.Navigate(consumURL),
		chromedp.WaitVisible(".page_content", chromedp.ByQuery),
		chromedp.Sleep(400*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("consumables page navigate: %w", err)
	}

	results := make(map[string]*GuideData)

	if heroSpec.ConsumSubTabsEnabled {
		// Nested sub-tab path: per-hero gems/enchants/consumables.
		for _, tab := range heroSpec.Tabs {
			stats, ok := heroStats[tab.Key]
			if !ok {
				continue
			}

			if tab.ConsumHeroSelector != "" {
				if err := clickTab(consumCtx, tab.ConsumHeroSelector); err != nil {
					fmt.Printf("WARN [%s] consum hero tab failed for %s (%s): %v\n", tag, tab.Key, tab.ConsumHeroSelector, err)
					continue
				}
			}

			guide := &GuideData{
				Class:   class,
				Spec:    spec,
				Updated: today,
				Stats:   stats,
			}

			for _, sub := range tab.ConsumSubTabs {
				if err := clickTab(consumCtx, sub.Selector); err != nil {
					fmt.Printf("WARN [%s] sub-tab click failed for %s %s (%s): %v\n", tag, tab.Key, sub.Content, sub.Selector, err)
					continue
				}
				text, err := readPageText(consumCtx)
				if err != nil {
					fmt.Printf("WARN [%s] sub-tab read failed for %s %s: %v\n", tag, tab.Key, sub.Content, err)
					continue
				}
				switch sub.Content {
				case "gems":
					guide.Gems = parseGems(text)
				case "enchants":
					guide.Enchants = parseEnchants(text)
				case "consumables":
					guide.Consumables = parseConsumables(text)
				}
			}

			// Fail loudly if sub-tab scraping produced empty data.
			if guide.Gems == nil || len(guide.Enchants) == 0 {
				fmt.Printf("WARN [%s] %s: sub-tab scrape produced empty gems or enchants — skipping hero key\n", tag, tab.Key)
				continue
			}

			results[tab.Key] = guide
		}
	} else {
		// Single-read path: parse once, share across all hero keys.
		consumText, err := readPageText(consumCtx)
		if err != nil {
			return nil, fmt.Errorf("consumables page read: %w", err)
		}

		sharedGems := parseGems(consumText)
		sharedEnchants := parseEnchants(consumText)
		sharedConsum := parseConsumables(consumText)

		for key, stats := range heroStats {
			results[key] = &GuideData{
				Class:       class,
				Spec:        spec,
				Updated:     today,
				Stats:       stats,
				Gems:        sharedGems,
				Enchants:    sharedEnchants,
				Consumables: sharedConsum,
			}
		}

		// warnGuideGaps once (consumables are shared).
		for _, guide := range results {
			warnGuideGaps(consumText, guide, class, spec)
			break
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no hero talent guide data assembled")
	}

	return results, nil
}

// normalizeStatLine checks whether a single line from a stat priority page
// contains one or more known stat names. It returns the canonical stat names
// found, or nil if the line doesn't parse as stats.
//
// Design principle: Icy Veins expresses "approximately equal" stats in many
// formats — "A = B", "A >= B", "A ≅ B", "A and B", "A, B or C". These are
// all equality indicators. We normalize them to a single separator before
// matching against knownStats, rather than handling each variant individually.
func normalizeStatLine(line string, knownStats []string) []string {
	if len(line) == 0 || len(line) > 80 {
		return nil
	}

	// Fast path: exact match for a single known stat.
	for _, stat := range knownStats {
		if strings.EqualFold(line, stat) {
			return []string{stat}
		}
	}

	// Normalize all equality indicators to "=".
	normalized := line
	for _, eq := range []string{"≅", "≈", "≃", "≥", "~="} {
		normalized = strings.ReplaceAll(normalized, eq, "=")
	}
	normalized = strings.ReplaceAll(normalized, ">=", "=")
	normalized = strings.ReplaceAll(normalized, "<=", "=")
	normalized = strings.ReplaceAll(normalized, "/", "=")

	// Split by =, comma, and &.
	parts := regexp.MustCompile(`[=,&]`).Split(normalized, -1)

	// Further split each part by word-boundary "and" / "or".
	var tokens []string
	wordSep := regexp.MustCompile(`(?i)\band\b|\bor\b`)
	for _, p := range parts {
		for _, s := range wordSep.Split(p, -1) {
			// Strip parenthetical suffixes like "(25% - 33%)".
			if idx := strings.Index(s, "("); idx > 0 {
				s = s[:idx]
			}
			s = strings.TrimSpace(s)
			s = strings.TrimRight(s, "*") // strip footnote markers (e.g., "Haste*")
			s = strings.TrimSpace(s)
			if s != "" {
				tokens = append(tokens, s)
			}
		}
	}

	if len(tokens) == 0 {
		return nil
	}

	// Stat name aliases: short forms used on some Icy Veins pages.
	statAliases := map[string]string{
		"crit": "Critical Strike",
	}

	// Every token must be a known stat; use the canonical name from knownStats.
	var matched []string
	for _, token := range tokens {
		found := false
		for _, stat := range knownStats {
			if strings.EqualFold(token, stat) {
				matched = append(matched, stat)
				found = true
				break
			}
		}
		if !found {
			if canonical, ok := statAliases[strings.ToLower(token)]; ok {
				matched = append(matched, canonical)
				found = true
			}
		}
		if !found {
			return nil
		}
	}
	return matched
}

// Convention-based regexes for warnGuideGaps. These detect item names by
// naming pattern (title-cased words + category suffix) rather than by
// allowlist, so they can catch items missing from the allowlist.
//
// Known limitations:
//   - Food has no extractable naming convention (names are proper nouns).
//   - Weapon buffs like "Crystalline Radiance" don't end in Oil/Whetstone.
//   Both categories rely on allowlist-only detection for those outliers.
var (
	warnHeadPotionRe = regexp.MustCompile(`(?i)\bpotion`)
	warnHeadFoodRe   = regexp.MustCompile(`(?i)\bfood`)
	warnFlaskRe      = regexp.MustCompile(`Flask of (?:the |)(?:[A-Z][a-z'-]+ ){0,3}[A-Z][a-z'-]+`)
	warnPotionRe     = regexp.MustCompile(`(?:Potion|Draught) of (?:the |)(?:[A-Z][a-z'-]+ ){0,3}[A-Z][a-z'-]+`)
	warnOilRe        = regexp.MustCompile(`(?:[A-Z][a-z'-]+ )+(?:Oil|Whetstone)`)
	warnRuneRe       = regexp.MustCompile(`[A-Z][\w'-]+ Augment Rune`)
	warnGemRe          = regexp.MustCompile(`(?:Flawless|Perfect) \w+ (?:Diamond|Amethyst|Garnet|Peridot|Lapis|Ruby|Sapphire)|\w+ Eversong Diamond`)
	warnEnchantItemRe  = regexp.MustCompile(`Enchant (?:Helm|Chest|Shoulders|Ring|Boots|Weapon|Cloak|Bracers) - `)
)

// warnGuideGaps logs warnings when parsed guide data has empty fields that
// appear to be gaps (page covers the topic but no item matched) rather than
// legitimate absences (page doesn't cover the topic).
//
// Per consumable category, detection is two-tier:
//  1. Sub-heading found + known allowlist item in span but field empty →
//     parsing/extraction bug.
//  2. Sub-heading found + convention regex matches an item NOT in the allowlist →
//     allowlist is stale.
//  3. Sub-heading found but neither known items nor convention matches in span →
//     legitimate absence (heading explains why the category doesn't apply).
//
// For gems and enchants, warnings fire when parsing returned empty but the
// page text contains convention-based patterns indicating content was present.
//
// Known limitations:
//   - Specs with inline-only consumables (no sub-headings, e.g., Resto Druid)
//     won't trigger warnings. Deferred to Step 5 (pattern extraction).
//   - Food has no convention regex; only allowlist detection (tier 1) applies.
//   - Weapon buffs not ending in Oil/Whetstone are invisible to tier 2.
func warnGuideGaps(consumText string, guide *GuideData, class, spec string) {
	tag := class + "_" + spec

	// --- Consumable category checks ---
	type consumCheck struct {
		name   string
		value  string
		isHead func(string) bool
		known  []string
		itemRe *regexp.Regexp // convention regex; nil = no pattern (food)
	}

	checks := []consumCheck{
		{
			name: "flask", value: guide.Consumables.Flask,
			isHead: func(low string) bool {
				return strings.HasPrefix(low, "flask") || strings.HasPrefix(low, "best flask")
			},
			known: knownConsumables["flask"], itemRe: warnFlaskRe,
		},
		{
			name: "potion", value: guide.Consumables.Potion,
			isHead: func(low string) bool { return warnHeadPotionRe.MatchString(low) },
			known:  knownConsumables["potion"], itemRe: warnPotionRe,
		},
		{
			name: "food", value: guide.Consumables.Food,
			isHead: func(low string) bool { return warnHeadFoodRe.MatchString(low) },
			known:  knownConsumables["food"], itemRe: nil,
		},
		{
			name: "weaponOil", value: guide.Consumables.WeaponOil,
			isHead: func(low string) bool {
				return strings.HasPrefix(low, "weapon buff") ||
					strings.HasPrefix(low, "weapon enhancement") ||
					strings.HasPrefix(low, "weapon oil")
			},
			known: knownConsumables["weaponOil"], itemRe: warnOilRe,
		},
		{
			name: "augRune", value: guide.Consumables.AugRune,
			isHead: func(low string) bool { return strings.HasPrefix(low, "augment") },
			known:  knownConsumables["augRune"], itemRe: warnRuneRe,
		},
	}

	// All heading matchers, for span-boundary detection.
	allHeadMatchers := make([]func(string) bool, len(checks))
	for i, c := range checks {
		allHeadMatchers[i] = c.isHead
	}
	isAnyHead := func(low string) bool {
		for _, fn := range allHeadMatchers {
			if fn(low) {
				return true
			}
		}
		return false
	}

	lines := strings.Split(consumText, "\n")
	for _, chk := range checks {
		if chk.value != "" {
			continue
		}

		// Find sub-heading line (short line matching this category).
		headLine := -1
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) == 0 || len(trimmed) >= 60 {
				continue
			}
			if chk.isHead(strings.ToLower(trimmed)) {
				headLine = i
				break
			}
		}
		if headLine < 0 {
			continue // no sub-heading — legitimate absence
		}

		// Extract span: sub-heading to next sub-heading, capped at 300 chars.
		var spanBuilder strings.Builder
		charCount := 0
		for i := headLine + 1; i < len(lines) && charCount < 300; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed != "" && len(trimmed) < 60 && i > headLine+1 {
				if isAnyHead(strings.ToLower(trimmed)) {
					break
				}
			}
			spanBuilder.WriteString(lines[i])
			spanBuilder.WriteString("\n")
			charCount += len(lines[i]) + 1
		}
		span := spanBuilder.String()
		spanLower := strings.ToLower(span)

		// Tier 1: known item in span but field empty → parsing/extraction bug.
		tier1 := false
		for _, item := range chk.known {
			if strings.Contains(spanLower, strings.ToLower(item)) {
				fmt.Printf("WARN [%s] %s: known item %q found on page but missing from parsed output\n", tag, chk.name, item)
				tier1 = true
				break
			}
		}

		// Tier 2: convention regex matches unknown item → allowlist stale.
		if !tier1 && chk.itemRe != nil {
			match := chk.itemRe.FindString(span)
			if match != "" {
				inKnown := false
				for _, item := range chk.known {
					if strings.EqualFold(item, match) {
						inKnown = true
						break
					}
				}
				if !inKnown {
					fmt.Printf("WARN [%s] %s: page mentions %q — not in known list\n", tag, chk.name, match)
				}
			}
		}
	}

	// Gems: if parseGems returned nil, check whether the page has gem content
	// that section extraction missed. Uses convention regex (not allowlist) to
	// avoid circular detection.
	if guide.Gems == nil {
		if warnGemRe.MatchString(consumText) {
			fmt.Printf("WARN [%s] gems: section heading not found but page contains gem-name patterns\n", tag)
		}
	}

	// Enchants: if parseEnchants returned empty, check whether the page has
	// enchant item names (not just the word "enchant" in prose). Pattern matches
	// the WoW enchant naming convention: "Enchant {Slot} - {Name}".
	if len(guide.Enchants) == 0 {
		if warnEnchantItemRe.MatchString(consumText) {
			fmt.Printf("WARN [%s] enchants: page contains enchant items but none were parsed\n", tag)
		}
	}
}

func parseStatPriority(text string) StatPriority {
	sp := StatPriority{}

	knownStats := []string{
		"Agility", "Strength", "Intellect", "Stamina",
		"Haste", "Mastery", "Critical Strike", "Versatility",
		"Item Level", "Armor",
	}

	// Path 1: percentage format ("50% into Mastery"). Convert to ordered
	// by sorting descending by percentage. Ties preserve page order.
	pctRegex := regexp.MustCompile(`(\d+)%\s+into\s+([A-Za-z ]+)`)
	pctMatches := pctRegex.FindAllStringSubmatch(text, -1)
	if len(pctMatches) >= 2 {
		type pctEntry struct {
			stat string
			pct  int
		}
		var entries []pctEntry
		for _, m := range pctMatches {
			pct, _ := strconv.Atoi(m[1])
			stat := strings.TrimSpace(m[2])
			stat = strings.TrimSuffix(stat, " FAQ")
			stat = strings.TrimSuffix(stat, " Gems")
			stat = strings.TrimSuffix(stat, " The")
			entries = append(entries, pctEntry{stat: stat, pct: pct})
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].pct > entries[j].pct
		})
		sp.Format = "ordered"
		for _, e := range entries {
			sp.Ordered = append(sp.Ordered, e.stat)
		}
		return sp
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Count(line, ">") >= 2 && len(line) < 150 {
			sp.Format = "ordered"
			parts := strings.Split(line, ">")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" || len(p) >= 45 {
					continue
				}
				// Expand compound chunks like "Crit = Mastery = Vers"
				// through normalizeStatLine, which handles =, /, &, etc.
				if expanded := normalizeStatLine(p, knownStats); len(expanded) > 0 {
					sp.Ordered = append(sp.Ordered, expanded...)
				} else {
					sp.Ordered = append(sp.Ordered, p)
				}
			}
			if len(sp.Ordered) >= 2 {
				return sp
			}
			sp.Ordered = nil
		}
	}

	// Path 3: consecutive lines that are known stats, with equality
	// normalization for formats like "A = B", "A >= B", "A ≅ B",
	// "A and B", "A, B or C".
	// TODO v0.3.1: preserve tied-stat semantic instead of flattening
	// to adjacent entries in Ordered.
	var consecutiveStats []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		stats := normalizeStatLine(line, knownStats)
		if len(stats) > 0 {
			consecutiveStats = append(consecutiveStats, stats...)
		} else if len(consecutiveStats) >= 2 {
			break
		} else {
			consecutiveStats = nil
		}
	}
	if len(consecutiveStats) >= 2 {
		sp.Format = "ordered"
		sp.Ordered = consecutiveStats
		return sp
	}

	orderedRegex := regexp.MustCompile(`(?i)stat priority[^:]*:\s*\n?((?:[A-Za-z ]+(?:=|>|\n)){2,})`)
	if m := orderedRegex.FindStringSubmatch(text); m != nil {
		sp.Format = "ordered"
		parts := regexp.MustCompile(`[>\n]`).Split(m[1], -1)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, "=")
			if p != "" && len(p) < 30 {
				sp.Ordered = append(sp.Ordered, p)
			}
		}
		if len(sp.Ordered) >= 2 {
			return sp
		}
	}

	sp.Format = "ordered"
	sp.Note = extractStatNote(text)
	return sp
}

func extractStatNote(text string) string {
	sentences := strings.Split(text, ".")
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		statWords := []string{"Haste", "Mastery", "Critical Strike", "Versatility", "Crit"}
		count := 0
		for _, w := range statWords {
			if strings.Contains(s, w) {
				count++
			}
		}
		if count >= 2 && len(s) < 200 {
			return s
		}
	}
	return ""
}

var knownGems = []string{
	"Indecipherable Eversong Diamond",
	"Powerful Eversong Diamond",
	"Stoic Eversong Diamond",
	"Telluric Eversong Diamond",
	"Thalassian Diamond",
	"Flawless Deadly Amethyst",
	"Flawless Quick Amethyst",
	"Flawless Masterful Amethyst",
	"Flawless Versatile Amethyst",
	"Flawless Deadly Lapis",
	"Flawless Quick Lapis",
	"Flawless Masterful Lapis",
	"Flawless Versatile Lapis",
	"Flawless Deadly Peridot",
	"Flawless Quick Peridot",
	"Flawless Masterful Peridot",
	"Flawless Versatile Peridot",
	"Flawless Quick Garnet",
	"Flawless Masterful Garnet",
}

func parseGems(text string) []GemEntry {
	section := extractSectionByHeading(text,
		[]string{"Recommended Gems", "Best Gems", "Gems for", "Oracle Gems", "Voidweaver Gems"},
		[]string{"Enchants for", "Best Enchants for", "Best Enchants", "Enchants and", "Oracle Enchants", "Voidweaver Enchants", "Changelog"},
	)
	if section == "" {
		return nil
	}

	section = regexp.MustCompile(`\s{2,}`).ReplaceAllString(section, " ")

	var gems []GemEntry
	seen := map[string]bool{}
	sectionLower := strings.ToLower(section)

	for _, name := range knownGems {
		if strings.Contains(sectionLower, strings.ToLower(name)) && !seen[name] {
			seen[name] = true
			gems = append(gems, GemEntry{Name: name})
		}
	}
	return gems
}

func parseEnchants(text string) []EnchantEntry {
	var enchants []EnchantEntry

	section := extractSectionByHeading(text,
		[]string{"Enchants for", "Best Enchants for", "Best Enchants", "Oracle Enchants", "Voidweaver Enchants"},
		[]string{"Consumable Recommendation", "Extra Consumable", "Best Midnight Consumables", "Best Consumables", "Consumables for", "Oracle Consumables", "Voidweaver Consumables", "How to Choose", "Changelog"},
	)
	if section == "" {
		return enchants
	}

	slotNames := []string{
		"Weapon", "Off Hand", "Rings", "Ring", "Helmet", "Helm", "Head",
		"Shoulder", "Shoulders", "Chest", "Legs", "Boots", "Feet",
		"Cloak", "Back", "Bracers", "Wrist", "Waist", "Hands",
	}

	enchantPattern := regexp.MustCompile(`Enchant [A-Za-z]+ - [A-Za-z` + "`" + `' ]+`)
	trailingNoise := regexp.MustCompile(`\s+(until|or |when |as |if |\(|,|and )`)

	extractEnchant := func(s string) string {
		s = regexp.MustCompile(`\s{2,}`).ReplaceAllString(s, " ")
		s = strings.TrimSpace(s)
		if m := enchantPattern.FindString(s); m != "" {
			m = strings.TrimSpace(m)
			if idx := trailingNoise.FindStringIndex(m); idx != nil {
				m = strings.TrimSpace(m[:idx[0]])
			}
			return m
		}
		if strings.Contains(s, "Kit") || strings.Contains(s, "Spellthread") {
			if idx := strings.Index(s, " ("); idx > 0 {
				s = s[:idx]
			}
			return strings.TrimSpace(s)
		}
		return ""
	}

	normaliseSlot := func(raw string) string {
		raw = strings.TrimSpace(raw)
		switch strings.ToLower(raw) {
		case "head":
			return "Helm"
		case "shoulder":
			return "Shoulders"
		case "ring":
			return "Rings"
		case "feet":
			return "Boots"
		case "off hand":
			return "Off Hand"
		}
		return raw
	}

	seen := map[string]bool{}
	addEnchant := func(slot, enchant string) {
		slot = normaliseSlot(slot)
		key := slot + "|" + enchant
		if !seen[key] && enchant != "" {
			seen[key] = true
			enchants = append(enchants, EnchantEntry{Slot: slot, Enchant: enchant})
		}
	}

	lines := strings.Split(section, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				slot := strings.TrimSpace(parts[0])
				enchantRaw := strings.TrimSpace(parts[1])
				isSlot := false
				for _, s := range slotNames {
					if strings.EqualFold(slot, s) {
						isSlot = true
						break
					}
				}
				if isSlot {
					if enc := extractEnchant(enchantRaw); enc != "" {
						addEnchant(slot, enc)
					}
				}
			}
			continue
		}

		for _, slot := range slotNames {
			if !strings.EqualFold(line, slot) {
				continue
			}
			for j := i + 1; j < len(lines) && j < i+8; j++ {
				next := strings.TrimSpace(lines[j])
				if next == "" {
					continue
				}
				isNextSlot := false
				for _, s2 := range slotNames {
					if strings.EqualFold(next, s2) {
						isNextSlot = true
						break
					}
				}
				if isNextSlot {
					break
				}
				if enc := extractEnchant(next); enc != "" {
					addEnchant(slot, enc)
					break
				}
			}
			break
		}
	}
	return enchants
}

var knownConsumables = map[string][]string{
	"flask": {
		"Flask of the Magisters",
		"Flask of the Shattered Sun",
		"Flask of the Blood Knights",
		"Flask of Thalassian Resistance",
		"Flask of Crystallized Speed",
		"Flask of Tempered Swiftness",
		"Flask of Tempered Versatility",
		"Flask of Tempered Aggression",
		"Flask of Tempered Mastery",
	},
	"potion": {
		"Light's Potential",
		"Potion of Recklessness",
		"Draught of Rampant Abandon",
		"Potion of Shocking Disclosure",
		"Tempered Potion",
		"Algari Healing Potion",
		"Silvermoon Health Potion",
	},
	"food": {
		"Silvermoon Parade",
		"Harandar Celebration",
		"Blooming Feast",
		"Quel'dorei Medley",
		"Royal Roast",
		"Impossibly Royal Roast",
		"Champion's Bento",
		"Ren'dorei Banquet",
		"Sinner's Stew",
	},
	"weaponOil": {
		"Thalassian Phoenix Oil",
		"Algari Mana Oil",
		"Mercurial Whetstone",
		"Ironclaw Whetstone",
		"Crystalline Radiance",
	},
	"augRune": {
		"Void-Touched Augment Rune",
	},
}

func parseConsumables(text string) ConsumableSet {
	cs := ConsumableSet{}

	consumBlock := extractSectionByHeading(text,
		[]string{"Consumable Recommendations", "Extra Consumable", "Best Midnight Consumables", "Best Consumables", "Consumables for", "Oracle Consumables", "Voidweaver Consumables", "Best Flasks and Potions"},
		[]string{"Changelog"},
	)
	if consumBlock == "" {
		consumBlock = text
	}

	consumBlock = regexp.MustCompile(`\s{2,}`).ReplaceAllString(consumBlock, " ")

	cs.Flask = findFirstKnown(consumBlock, knownConsumables["flask"])
	cs.Potion = findFirstKnown(consumBlock, knownConsumables["potion"])
	cs.Food = findFirstKnown(consumBlock, knownConsumables["food"])
	cs.WeaponOil = findFirstKnown(consumBlock, knownConsumables["weaponOil"])
	cs.AugRune = findFirstKnown(consumBlock, knownConsumables["augRune"])

	if cs.WeaponOil == "" {
		fullNorm := regexp.MustCompile(`\s{2,}`).ReplaceAllString(text, " ")
		cs.WeaponOil = findFirstKnown(fullNorm, knownConsumables["weaponOil"])
	}

	return cs
}

func findFirstKnown(text string, candidates []string) string {
	lowerText := strings.ToLower(text)
	bestIdx := -1
	bestName := ""
	for _, name := range candidates {
		idx := strings.Index(lowerText, strings.ToLower(name))
		if idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
			bestIdx = idx
			bestName = name
		}
	}
	return bestName
}

func extractSectionByHeading(text string, startMarkers, endMarkers []string) string {
	lines := strings.Split(text, "\n")

	startLine := -1
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, marker := range startMarkers {
			if strings.EqualFold(line, marker) ||
				strings.HasPrefix(strings.ToLower(line), strings.ToLower(marker)) {
				startLine = i + 1
				break
			}
		}
		if startLine >= 0 {
			break
		}
	}
	if startLine < 0 {
		return ""
	}

	endLine := len(lines)
	for i := startLine; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		for _, marker := range endMarkers {
			if strings.EqualFold(line, marker) ||
				strings.HasPrefix(strings.ToLower(line), strings.ToLower(marker)) {
				endLine = i
				break
			}
		}
		if endLine < len(lines) {
			break
		}
	}

	section := strings.TrimSpace(strings.Join(lines[startLine:endLine], "\n"))
	return section
}

func generateGuide() error {
	files, err := filepath.Glob("data/guide_*.json")
	if err != nil || len(files) == 0 {
		return fmt.Errorf("no guide JSON files found — run 'go run . scrape-guide' first")
	}

	sort.Strings(files)

	// Parse filenames: guide_CLASS_Spec_HeroKey.json
	// The hero key is everything after the second underscore in the base name
	// (after stripping the "guide_" prefix and ".json" suffix).
	type parsedFile struct {
		Class   string
		Spec    string
		HeroKey string
		Data    GuideData
	}

	var parsed []parsedFile
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", f, err)
		}
		var g GuideData
		if err := json.Unmarshal(raw, &g); err != nil {
			return fmt.Errorf("error parsing %s: %w", f, err)
		}

		// Extract hero key from filename.
		base := filepath.Base(f)                           // guide_CLASS_Spec_HeroKey.json
		base = strings.TrimPrefix(base, "guide_")          // CLASS_Spec_HeroKey.json
		base = strings.TrimSuffix(base, ".json")           // CLASS_Spec_HeroKey
		parts := strings.SplitN(base, "_", 3)              // [CLASS, Spec, HeroKey]
		if len(parts) < 3 {
			return fmt.Errorf("unexpected filename format: %s (expected guide_CLASS_Spec_HeroKey.json)", f)
		}

		parsed = append(parsed, parsedFile{
			Class:   parts[0],
			Spec:    parts[1],
			HeroKey: parts[2],
			Data:    g,
		})
		fmt.Printf("  Loaded: %s %s [%s]\n", parts[0], parts[1], parts[2])
	}

	// Group by class+spec.
	groupMap := make(map[string]*GuideGroup)
	var groupOrder []string
	for _, p := range parsed {
		key := p.Class + "_" + p.Spec
		grp, exists := groupMap[key]
		if !exists {
			grp = &GuideGroup{Class: p.Class, Spec: p.Spec}
			groupMap[key] = grp
			groupOrder = append(groupOrder, key)
		}
		grp.Entries = append(grp.Entries, GuideEntry{
			HeroKey:     p.HeroKey,
			Updated:     p.Data.Updated,
			Stats:       p.Data.Stats,
			Gems:        p.Data.Gems,
			Enchants:    p.Data.Enchants,
			Consumables: p.Data.Consumables,
		})
	}

	var groups []GuideGroup
	for _, key := range groupOrder {
		groups = append(groups, *groupMap[key])
	}

	tmpl, err := template.ParseFiles("templates/GearPath_Stats.lua.tmpl")
	if err != nil {
		return fmt.Errorf("could not load template: %w", err)
	}

	outputPath := filepath.Join("..", "Data", "GearPath_Stats.lua")
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create output file: %w", err)
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, GuideTemplateData{
		Generated: time.Now().Format("2006-01-02 15:04:05"),
		Groups:    groups,
	}); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	fmt.Printf("\nGenerated: %s (%d groups, %d entries)\n", outputPath, len(groups), len(parsed))
	return nil
}