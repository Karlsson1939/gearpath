# Changelog

All notable changes to GearPath will be documented in this file.

## [0.4.3] - 2026-04-26

### Fixed

- **Priority tab sub-row overlap.** Items shown when expanding a source row no longer overflow into rows below. Sub-rows are now anchored to a fixed point inside their parent row instead of a moving anchor that shifted on expand.

- **Chevron rendering on Priority tab rows.** The expand/collapse indicator in each row's bottom-right corner was rendering as an empty box because the Unicode triangle characters (▼ ▲) aren't supported by the addon's body font. Replaced with WoW atlas chevron textures that render correctly and indicate expand/collapse direction.

## [0.4.2] - 2026-04-26

### Fixed

- **Raid items correctly classified.** Items from The Voidspire, The Dreamrift, and March on Quel'Danas now show as "Raid" sources instead of "Dungeon" — they were previously miscategorized due to a Blizzard journal-instance API quirk. Fixed by overriding the instance type at scrape and generate time.

## [0.4.1] - 2026-04-26

### Fixed

- **Vault tab crash.** Opening the Vault tab no longer triggers a Lua error. The progress bar resize handler was incorrectly attached to a Texture object; moved to the parent Frame which supports the script.

- **Priority tab sub-row overlap.** Expanded sub-row content no longer bleeds through into other source rows. Sub-rows are now correctly parented to their owning source row, so they hide and refresh together.

### Added

- **Addon icon in the AddOns list.** GearPath now shows the gear icon instead of the default red questionmark.

## [0.4.0] - 2026-04-26

### Added

- **Design token system.** A central Theme module defines colors, fonts, spacing, and borders used consistently across the addon. The main frame is now framed by a 2px outer border in your class color, and class color also accents the spec label, active tab, and BiS completion progress bar.

- **Status icon legend on the BiS List tab.** A row at the top of the tab explains what the ✓ B ↑ ✗ icons mean.

- **Tab button tooltips.** Hovering a tab in the main panel shows a one-line description of what the tab contains.

- **Score explanation on the Priority tab.** A muted line below the BiS completion bar describes how sources are ranked.

- **Data source attribution on the Stats tab.** A footer credits Icy Veins as the source of stat priorities, gems, enchants, and consumables.

### Fixed

- **Priority tab expand/collapse.** Clicking a source row to expand it no longer causes layout overlap or fails to collapse on a second click.

- **Vault tab auto-refresh.** The Vault tab now updates in real time when weekly progress changes (e.g., after completing a Mythic+ key), instead of only refreshing when the tab is reopened.

- **No more chat spam on detection.** The addon no longer prints "[Detection] Detected: ..." messages on login, spec change, or zone-in.

- **BiS List empty state.** When no BiS data is available for the current spec, the tab now shows a clear message instead of rendering blank.

- **Source type names display correctly.** "Dungeon", "Raid", "PvP" etc. instead of raw enum strings.

- **Section headers are consistently title-cased** across all tabs.

- **`/gp` help text now describes each command.**

- **Detection-in-progress message** now suggests `/reload` if detection doesn't complete naturally.

### Removed

- **Config stub.** The Config.lua placeholder, the right-click handler on the minimap button, the `/gp config` slash command, and the misleading "Right-click for settings" tooltip line have all been removed. They referenced functionality that didn't exist. A real settings UI is planned for a future release.

## [0.3.1] - 2026-04-26

### Added

- **Full 40-spec coverage.** Stats & Consumables data now covers every spec in the game, up from 13 in v0.3.0.

- **Discipline Priest per-hero-talent data.** Oracle and Voidweaver have separate gems, enchants, and consumables sourced from Icy Veins' nested tab UI.

### Fixed

- **Stat priorities for 5 specs.** Marksmanship Hunter, Vengeance Demon Hunter, Protection Paladin, Brewmaster Monk, and Discipline Priest now show complete stat priority lists. Previously these specs showed only 0-2 stats due to unhandled formatting on Icy Veins (slash separators, ampersand separators, footnote markers, and abbreviated stat names).

- **Brewmaster Monk augment rune.** Was silently dropped because the consumable section text exceeded an internal length limit. The limit has been removed; consumable sections are now bounded only by their start and end markers.

## [0.3.0] - 2026-04-18

### Added

- **Stats & Consumables tab.** New tab showing stat priority, recommended gems, enchants, and consumables for your current spec and hero talent. Data updates automatically when you switch hero talents.

- **Per-hero-talent stat priorities.** Blood DK (Deathbringer / San'layn), Enhancement Shaman (Stormbringer / Totemic), Shadow Priest (Archon / Voidweaver), and Holy Paladin (Herald of the Sun / Lightsmith) show correct stat priorities for each hero talent. Holy Paladin includes a "General" stat priority as fallback for players without an active hero talent selection.

- Stats & Consumables data covers 13 specs across all classes. Full 40-spec coverage is in progress.

### Changed

- Tab order is now Priority / BiS List / Stats / Vault. Vault moved from position 3 to 4.

## [0.2.6] - 2026-04-17

### Fixed

- BiS recommendations now identify the correct items across all 13 classes. Previously, many tier pieces and crafted items had wrong internal IDs, causing the addon to miss equipped BiS items or track incorrect ones.

- Affected: Death Knight, Shaman, Mage, Warlock, Evoker, and Warrior tier sets; crafted items across Blacksmithing, Leatherworking, Tailoring, and Jewelcrafting; several dungeon drops.

- Platinum Star Band now correctly attributed to Algeth'ar Academy (previously mislabeled as crafted).

- Recognition of "Scabrous Zombie Belt" — previously dropped silently due to a naming mismatch between in-game and the scraped BiS data.
