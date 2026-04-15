# GearPath

A World of Warcraft addon for Midnight Season 1 that ranks content sources by how many BiS upgrades they contain for your class and spec.

## Supported specs (Season 1)

| Class | Specs |
|-------|-------|
| Death Knight | Blood |
| Demon Hunter | Devourer |
| Druid | Guardian |
| Hunter | Survival |
| Mage | Frost |
| Shaman | Elemental |

More specs coming — contributions welcome via GitHub.

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
