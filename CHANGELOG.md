# Changelog

All notable changes to GearPath will be documented in this file.

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
