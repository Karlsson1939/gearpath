# GearPath

> ⚠️ **Work in progress** — GearPath is in active early development. Features are incomplete, data only covers a subset of specs, and bugs are expected. Feedback and contributions welcome via GitHub Issues.

A World of Warcraft addon for Midnight Season 1 that ranks content sources by how many BiS upgrades they contain for your class and spec, and shows stat priorities, gems, enchants, and consumables.

## Data sources

GearPath's BiS recommendations, stat priorities, gems, enchants, and consumable data are sourced from [Icy Veins](https://www.icy-veins.com/) guide pages. Data is scraped periodically by the companion app and may lag behind live Icy Veins updates by a few days.

This addon is not affiliated with or endorsed by Icy Veins.

## Supported specs (Season 1)

Stats & Consumables data covers all 40 specs. Hero-talent-specific stat priorities and consumables are noted where applicable.

| Class | Spec | Hero talents |
|-------|------|-------------|
| Death Knight | Blood | Deathbringer, San'layn |
| Death Knight | Frost | |
| Death Knight | Unholy | |
| Demon Hunter | Devourer | |
| Demon Hunter | Havoc | |
| Demon Hunter | Vengeance | |
| Druid | Balance | |
| Druid | Feral | |
| Druid | Guardian | |
| Druid | Restoration | |
| Evoker | Augmentation | |
| Evoker | Devastation | |
| Evoker | Preservation | |
| Hunter | Beast Mastery | |
| Hunter | Marksmanship | |
| Hunter | Survival | |
| Mage | Arcane | |
| Mage | Fire | |
| Mage | Frost | |
| Monk | Brewmaster | |
| Monk | Mistweaver | |
| Monk | Windwalker | |
| Paladin | Holy | Herald of the Sun, Lightsmith |
| Paladin | Protection | |
| Paladin | Retribution | |
| Priest | Discipline | Oracle, Voidweaver |
| Priest | Holy | |
| Priest | Shadow | Archon, Voidweaver |
| Rogue | Assassination | |
| Rogue | Outlaw | |
| Rogue | Subtlety | |
| Shaman | Elemental | |
| Shaman | Enhancement | Stormbringer, Totemic |
| Shaman | Restoration | |
| Warlock | Affliction | |
| Warlock | Demonology | |
| Warlock | Destruction | |
| Warrior | Arms | |
| Warrior | Fury | |
| Warrior | Protection | |

## Installation

1. Download from [CurseForge](https://www.curseforge.com/wow/addons/gearpath) or install via the CurseForge app.
2. Log in to a character in World of Warcraft: Midnight.
3. Type `/gp` to open the GearPath panel.

## Tabs

- **Priority** — Ranks content sources (raids, dungeons, crafted) by how many BiS upgrades remain.
- **BiS List** — Checklist of your BiS items by slot with equipped/missing status.
- **Stats** — Stat priorities, gems, enchants, and consumables for your spec and hero talent.
- **Vault** — Great Vault slot analysis showing which options are BiS upgrades.

## Structure

    GearPath.toc / *.lua   ← WoW addon (Lua 5.1)
    companion/             ← BiS data generator (Go, not distributed)

## Dev workflow

1. Edit `companion/data/CLASS_Spec.json`
2. `cd companion && go run . generate`
3. Commit the generated `Data/GearPath_Data.lua`
4. Tag a release to trigger CurseForge upload

## Slash commands

| Command | Action |
|---------|--------|
| `/gp` | Toggle main panel |
| `/gp priority` | Print top 3 sources to chat |
| `/gp bis` | Print gear summary to chat |
| `/gp vault` | Print vault summary to chat |
| `/gp reset` | Reset panel position |

## Known limitations (v0.3.1)

- **Hero-talent consumables are shared for some specs.** Blood DK (Deathbringer / San'layn) and Enhancement Shaman (Stormbringer / Totemic) have hero-specific consumable differences on Icy Veins that are embedded in prose rather than structured sections. Both hero talents currently show the same consumable recommendations. Discipline Priest (Oracle / Voidweaver) has per-hero consumables.
- **Shadow Priest Archon/Voidweaver stat priorities appear identical.** Icy Veins expresses the difference as percentage ranges (e.g., "Haste 25-33%" vs "24-29%"), which the addon doesn't capture — it only captures the ordering.
- **Holy Paladin Lightsmith weapon oil.** Both Herald of the Sun and Lightsmith show Thalassian Phoenix Oil. Lightsmith players using Rite of Sanctification may not need a weapon oil.
