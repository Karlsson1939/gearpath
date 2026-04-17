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

type TemplateData struct {
	Generated string
	Specs     []SpecData
}

// ============================================================
// Blizzard API structures
// ============================================================

type BlizzardToken struct {
	AccessToken string `json:"access_token"`
}

type BlizzardItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
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

// icyVeinsSlotMap maps every slot label variant to our internal slot ID.
// Pair-slot labels (ring, rings, trinket, trinkets, top trinkets) map to the
// FIRST slot of the pair; the parser uses pairFallback for the second.
var icyVeinsSlotMap = map[string]int{
	// ── Main hand / weapons ───────────────────────────────────────────────────
	"weapon":                       16,
	"weapons":                      16,
	"main hand":                    16,
	"mainhand weapon":              16,
	"one-handed weapon":            16,
	"1h weapon":                    16,
	"1h":                           16,
	"2h weapon":                    16,
	"2h":                           16,
	"weapon (2h)":                  16,
	"weapon (two-hand)":            16,
	"weapon (main-hand/off-hand)":  16,
	"weapon (staff)":               16,
	"mainhand 1h weapon":           16,
	"offhand 1h weapon":            17,
	"sentinel weapon":              16, // Hunter Survival 2H fallback (no hero talent support yet)
	"weapon main-hand":             16,
	"weapons (dual wield)":         16,
	"two-handed weapon":            16,
	"two-handed mace":              16,
	"staff":                        16,

	// ── Off hand ─────────────────────────────────────────────────────────────
	"off hand":        17,
	"off-hand":        17,
	"offhand":         17,
	"offhand weapon":  17,
	"off-hand weapon": 17,
	"weapon off-hand": 17,
	"shield":          17,

	// ── Armour ───────────────────────────────────────────────────────────────
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

	// ── Rings — numbered ─────────────────────────────────────────────────────
	"ring #1":   11,
	"ring #2":   12,
	"ring 1":    11,
	"ring 2":    12,
	"finger #1": 11,
	"finger #2": 12,
	"finger 1":  11,
	"finger 2":  12,

	// ── Rings — unnumbered / plural (first → 11, second → 12 via pairFallback)
	"ring":  11,
	"rings": 11,

	// ── Trinkets — numbered ───────────────────────────────────────────────────
	"trinket #1": 13,
	"trinket #2": 14,
	"trinket 1":  13,
	"trinket 2":  14,

	// ── Trinkets — unnumbered / plural (first → 13, second → 14) ─────────────
	"trinket":      13,
	"trinkets":     13,
	"top trinkets": 13, // multi-item cell: both trinkets listed; handled specially
}

// pairFallback maps a slot to its pair when the first is already taken.
var pairFallback = map[int]int{
	11: 12,
	13: 14,
}

// multiItemSlots: these label a cell that contains BOTH items of a pair,
// space-separated. We extract the first two and assign to slots 11+12 or 13+14.
var multiItemSlots = map[string]bool{
	"rings":        true,
	"top trinkets": true,
	"trinkets":     true, // single row listing both trinkets (e.g. Brewmaster, Holy Priest)
	"trinket":      true, // same pattern, singular label
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
		editSpec(reader, blizzard, s.Class, s.Spec)
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
	fmt.Println("Browser ready.\n")

	today := time.Now().Format("2006-01-02")
	fetched, failed, skipped := 0, 0, 0

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

		pageURL := fmt.Sprintf("https://www.icy-veins.com/wow/%s-gear-best-in-slot", slug)
		fmt.Printf("Scraping %-14s %-14s ... ", s.Class, s.Spec)

		pageCtx, pageCancel := context.WithTimeout(browserCtx, 30*time.Second)

		var tableData string
		err := chromedp.Run(pageCtx,
			chromedp.Navigate(pageURL),
			chromedp.WaitVisible(`table tr`, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Evaluate(`
				(function() {
					const t = document.querySelector('table');
					if (!t) return '';
					return Array.from(t.querySelectorAll('tr'))
						.map(r => Array.from(r.querySelectorAll('td,th'))
							.map(c => c.innerText.trim().replace(/\n+/g, ' '))
							.join('\t'))
						.join('\n');
				})()
			`, &tableData),
		)
		pageCancel()

		if err != nil {
			fmt.Printf("FAILED (browser: %v)\n", err)
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
			Class:   s.Class,
			Spec:    s.Spec,
			Season:  "Midnight S1",
			Updated: today,
			Items:   items,
		}

		filename := fmt.Sprintf("data/%s_%s.json", s.Class, s.Spec)
		data, _ := json.MarshalIndent(specData, "", "  ")
		if err := os.WriteFile(filename, data, 0644); err != nil {
			fmt.Printf("FAILED (write: %v)\n", err)
			failed++
			continue
		}

		fetched++
		time.Sleep(800 * time.Millisecond)
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

// parseIcyVeinsTable parses the tab-separated rows from the first BiS table.
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

		// ── Multi-item pair cells (Rings / Top Trinkets) ──────────────────────
		// These contain both items for a pair, space-separated. Extract first two.
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

		// ── Weapon cells with (1H) and (OH) markers ───────────────────────────
		// e.g. "Ceremonial Hexblade (1H)  Grimoire of the Eternal Light (OH)"
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

		// ── Normal single-slot row ────────────────────────────────────────────
		slotID, ok := icyVeinsSlotMap[slotKey]
		if !ok {
			continue
		}

		// Pair fallback for unlabelled repeated slots (Ring/Ring, Trinkets/Trinkets)
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
// Icy Veins separates them with 2+ spaces (collapsed from newlines).
func splitMultiItems(cell string) []string {
	// Split on two or more consecutive spaces
	parts := strings.FieldsFunc(cell, func(r rune) bool { return false })
	_ = parts

	// Use double-space as separator
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
	// Find the start of this item name by scanning backwards from the marker
	before := strings.TrimSpace(cell[:idx])
	// The item name is the last token before the marker (after any double-space separator)
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

// cleanItemName strips noise from item names:
//   - " with <embellishment>" suffixes
//   - "(TIER SET)", "(Crit/Haste)", "(1H)", "(OH)", "(2H)", "(Staff)" etc.
//   - ", or <alternative>" text
//   - Double-space separated alternatives (take first)
//   - Advisory rows starting with "Your "
func cleanItemName(name string) string {
	// Skip advisory rows
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "your ") {
		return ""
	}
	// Strip "BiS -> " navigation prefix — keep the actual item name after it
	trimmed := strings.TrimSpace(name)
	if strings.HasPrefix(strings.ToLower(trimmed), "bis -> ") {
		name = trimmed[7:]
	}
	// Strip " with ..." BEFORE double-space split (some cells have "Item with  Embellishment")
	if idx := strings.Index(name, " with "); idx > 0 {
		name = name[:idx]
	}
	// Strip ", or <alternative>"
	if idx := strings.Index(name, ", or"); idx > 0 {
		name = name[:idx]
	}
	// Strip trailing parenthetical suffixes like "(TIER SET)", "(1H)", "(Crit/Haste)"
	if idx := strings.Index(name, " ("); idx > 0 {
		name = name[:idx]
	}
	// Take only up to first double-space (multi-item cells, after above strips)
	if idx := strings.Index(name, "  "); idx > 0 {
		name = name[:idx]
	}
	// Strip newlines
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

	// ── Crafted ──────────────────────────────────────────────────────────────

	// Blacksmithing
	"Spellbreaker's Bracers":      {id: 237834, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Spellbreaker's Warglaive":    {id: 237840, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Blood Knight's Impetus":      {id: 237838, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Dawncrazed Beast Cleaver":    {id: 237836, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Mercy":          {id: 237839, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Chopper":        {id: 237837, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Farstrider's Plated Bracers": {id: 237835, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Bloomforged Claw":            {id: 237841, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Aln'hara Lantern":            {id: 245769, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Spellbreaker's Ultimatum":    {id: 237841, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Aln'hara Cane":               {id: 245770, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Leatherworking
	"World Tender's Barkclasp":        {id: 239650, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"World Tender's Rootslippers":     {id: 239652, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Sneakers":     {id: 244578, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Deflectors":   {id: 244576, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Utility Belt": {id: 244579, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Silvermoon Agent's Handwraps":    {id: 244575, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Tailoring
	"Arcanoweave Bracers":      {id: 239660, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Arcanoweave Cloak":        {id: 239661, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Arcanoweave Cord":         {id: 239662, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Adherent's Silken Shroud": {id: 239656, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Martyr's Bindings":        {id: 239663, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// Jewelcrafting
	"Platinum Star Band":          {id: 239658, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Loa Worshiper's Band":        {id: 239659, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Masterwork Sin'dorei Amulet": {id: 240950, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},
	"Masterwork Sin'dorei Band":   {id: 240949, sourceType: "CRAFTED", sourceName: "Crafted", source: "Crafted"},

	// ── March on Quel'Danas drops ────────────────────────────────────────────

	"Amulet of the Abyssal Hymn": {id: 250247, sourceType: "RAID", sourceName: "March on Quel'Danas", source: "marchonqueldanas", bossName: "Midnight Falls"},
	"Sin'dorei Band of Hope":     {id: 249919, sourceType: "RAID", sourceName: "March on Quel'Danas", source: "marchonqueldanas", bossName: "Child of Belo'ren"},

	// ── The Voidspire drops ──────────────────────────────────────────────────

	"Fallen King's Cuffs":           {id: 249304, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", bossName: "Fallen-King Salhadaar"},
	"Grimoire of the Eternal Light": {id: 249276, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", bossName: "Vorasius"},

	// ── Death Knight — Relentless Rider's Lament ─────────────────────────────

	"Relentless Rider's Crown":       {id: 249970, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Dreadthorns": {id: 249971, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Cuirass":     {id: 249973, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Bonegrasps":  {id: 249974, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Legguards":   {id: 249972, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Relentless Rider's Chain":       {id: 249969, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Demon Hunter — Devouring Reaver's Sheathe ────────────────────────────

	"Devouring Reaver's Intake":          {id: 250033, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Exhaustplates":   {id: 250031, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Engine":          {id: 250036, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Devouring Reaver's Essence Grips":   {id: 250034, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Pistons":         {id: 250032, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Devouring Reaver's Soul Flatteners": {id: 250035, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Druid — Sprouts of the Luminous Bloom ────────────────────────────────

	"Branches of the Luminous Bloom":     {id: 250024, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Seedpods of the Luminous Bloom":     {id: 250022, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Trunk of the Luminous Bloom":        {id: 250027, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Arbortenders of the Luminous Bloom": {id: 250025, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Phloemwraps of the Luminous Bloom":  {id: 250023, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Leafdrape of the Luminous Bloom":    {id: 250019, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Evoker — Livery of the Black Talon ───────────────────────────────────

	"Hornhelm of the Black Talon":         {id: 249997, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Beacons of the Black Talon":          {id: 249995, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Frenzyward of the Black Talon":       {id: 249993, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Enforcer's Grips of the Black Talon": {id: 249998, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Greaves of the Black Talon":          {id: 249996, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Spelltreads of the Black Talon":      {id: 249999, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Hunter — Primal Sentry's Camouflage ──────────────────────────────────

	"Primal Sentry's Maw":         {id: 249988, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Primal Sentry's Scaleplate":  {id: 249991, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Primal Sentry's Talonguards": {id: 249989, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Primal Sentry's Legguards":   {id: 249987, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// ── Mage — Voidbreaker's Accordance ──────────────────────────────────────

	"Voidbreaker's Veil":         {id: 249985, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Leyline Nexi": {id: 249983, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Robe":         {id: 249986, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Voidbreaker's Gloves":       {id: 250061, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Britches":     {id: 249984, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Voidbreaker's Sage Cord":    {id: 250057, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Voidbreaker's Treads":       {id: 249982, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Monk — Way of Ra-den's Chosen ────────────────────────────────────────

	"Fearsome Visage of Ra-den's Chosen": {id: 250015, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Aurastones of Ra-den's Chosen":      {id: 250013, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Battle Garb of Ra-den's Chosen":     {id: 250018, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Thunderfists of Ra-den's Chosen":    {id: 250016, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Swiftsweepers of Ra-den's Chosen":   {id: 250014, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Storm Crashers of Ra-den's Chosen":  {id: 250017, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Strikeguards of Ra-den's Chosen":    {id: 250019, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Windwrap of Ra-den's Chosen":        {id: 250020, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Paladin — Luminant Verdict's Vestments ───────────────────────────────

	"Luminant Verdict's Unwavering Gaze":  {id: 249961, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Providence Watch": {id: 249959, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Divine Warplate":  {id: 249964, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Luminant Verdict's Gauntlets":        {id: 249962, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Luminant Verdict's Greaves":          {id: 249960, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Aetherlume Stompers":                 {id: 251220, sourceType: "RAID", sourceName: "March on Quel'Danas", source: "marchonqueldanas", bossName: "Midnight Falls"},

	// ── Priest — Blind Oath's Burden ─────────────────────────────────────────

	"Blind Oath's Winged Crest": {id: 250051, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Seraphguards": {id: 250049, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Raiment":      {id: 250054, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Blind Oath's Leggings":     {id: 250050, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Touch":        {id: 250052, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blind Oath's Wraps":        {id: 250047, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Rogue — Motley of the Grim Jest ──────────────────────────────────────

	"Masquerade of the Grim Jest":       {id: 250006, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Venom Casks of the Grim Jest":      {id: 250004, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Fantastic Finery of the Grim Jest": {id: 250009, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Sleight of Hand of the Grim Jest":  {id: 250007, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Blade Holsters of the Grim Jest":   {id: 250005, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Balancing Boots of the Grim Jest":  {id: 250008, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// ── Shaman — Mantle of the Primal Core ───────────────────────────────────

	"Locus of the Primal Core":      {id: 249979, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Tempests of the Primal Core":   {id: 249980, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Embrace of the Primal Core":    {id: 249976, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Earthgrips of the Primal Core": {id: 249975, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Guardian of the Primal Core":   {id: 249965, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Ceinture of the Primal Core":   {id: 249977, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Leggings of the Primal Core":   {id: 249978, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},

	// ── Warlock — Reign of the Abyssal Immolator ─────────────────────────────

	"Abyssal Immolator's Smoldering Flames": {id: 250042, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Fury":              {id: 250046, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Dreadrobe":         {id: 250045, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Abyssal Immolator's Grasps":            {id: 250040, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Pillars":           {id: 250043, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Abyssal Immolator's Blazing Core":      {id: 250039, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},

	// Renamed via hotfix — Icy Veins still uses old name
	"Scabrous Zombie Leather Belt": {id: 49810, sourceType: "DUNGEON", sourceName: "Pit of Saron", source: "pitofsaron", bossName: "Ick"},

	"Night Ender's Tusks":       {id: 249952, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Pauldrons":   {id: 249950, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Breastplate": {id: 249955, sourceType: "RAID", sourceName: "The Dreamrift", source: "thedreamrift", isTier: true},
	"Night Ender's Chausses":    {id: 249951, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Fists":       {id: 249953, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: true},
	"Night Ender's Girdle":      {id: 249954, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
	"Night Ender's Greatboots":  {id: 249956, sourceType: "RAID", sourceName: "The Voidspire", source: "thevoidspire", isTier: false},
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
		if filepath.Base(f) == filepath.Base(sourceMapCachePath) {
			continue
		}
		spec, err := loadSpec(f)
		if err != nil {
			continue
		}
		if target != "" && spec.Class+"_"+spec.Spec != target {
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

func selectSpec(reader *bufio.Reader) *struct{ Class, Spec string } {
	fmt.Println("\nSelect a spec to edit:")
	fmt.Println()

	for i, s := range allSpecs {
		filename := fmt.Sprintf("data/%s_%s.json", s.Class, s.Spec)
		status := "empty"
		if spec, err := loadSpec(filename); err == nil && len(spec.Items) > 0 {
			status = fmt.Sprintf("%d/16 slots", len(spec.Items))
		}
		fmt.Printf("  %2d. %-14s %-14s [%s]\n", i+1, s.Class, s.Spec, status)
	}

	fmt.Println()
	fmt.Print("Enter number (q to quit): ")
	input := readLine(reader)

	if strings.ToLower(input) == "q" {
		return nil
	}

	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(allSpecs) {
		fmt.Println("Invalid selection.")
		return nil
	}

	s := allSpecs[n-1]
	return &struct{ Class, Spec string }{s.Class, s.Spec}
}

// ============================================================
// Spec editing
// ============================================================

func editSpec(reader *bufio.Reader, blizzard *BlizzardClient, class, spec string) {
	filename := fmt.Sprintf("data/%s_%s.json", class, spec)

	specData, err := loadSpec(filename)
	if err != nil {
		specData = &SpecData{
			Class:   class,
			Spec:    spec,
			Season:  "Midnight S1",
			Updated: time.Now().Format("2006-01-02"),
			Items:   []ItemData{},
		}
	}

	for {
		fmt.Printf("\n═══ %s %s ═══\n\n", specData.Class, specData.Spec)

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

	var specs []SpecData
	for _, file := range jsonFiles {
		if filepath.Base(file) == filepath.Base(sourceMapCachePath) {
			continue
		}
		spec, err := loadSpec(file)
		if err != nil {
			return fmt.Errorf("error parsing %s: %w", file, err)
		}
		if len(spec.Items) == 0 {
			fmt.Printf("  Skipping (empty): %s %s\n", spec.Class, spec.Spec)
			continue
		}
		sort.Slice(spec.Items, func(i, j int) bool {
			return spec.Items[i].SlotID < spec.Items[j].SlotID
		})
		specs = append(specs, *spec)
		fmt.Printf("  Loaded: %s %s (%d items)\n", spec.Class, spec.Spec, len(spec.Items))
	}

	if len(specs) == 0 {
		return fmt.Errorf("no specs with items found")
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
		Specs:     specs,
	}); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	fmt.Printf("\nGenerated: %s (%d specs)\n", outputPath, len(specs))
	return nil
}

// ============================================================
// Init-specs
// ============================================================

func initSpecs() error {
	today := time.Now().Format("2006-01-02")
	created, skipped := 0, 0

	for _, s := range allSpecs {
		filename := fmt.Sprintf("data/%s_%s.json", s.Class, s.Spec)
		if _, err := os.Stat(filename); err == nil {
			skipped++
			continue
		}
		skeleton := SpecData{
			Class:   s.Class,
			Spec:    s.Spec,
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