-- GearPath
-- UI/Theme.lua - Visual design tokens
-- Every UI file should reference these tokens rather than hardcoded values.
-- Class color is resolved at runtime in Theme:Init() since it's player-specific.

GearPath.Theme = {
    color = {
        -- Backgrounds (4 levels + bar track)
        bgBase     = {0.05, 0.05, 0.08, 0.95}, -- darkest, sub-rows
        bgSurface  = {0.08, 0.08, 0.12, 0.90}, -- standard rows
        bgElevated = {0.12, 0.12, 0.18, 0.95}, -- hover state
        bgDisabled = {0.06, 0.06, 0.08, 0.80}, -- locked rows
        bgTrack    = {0.0,  0.0,  0.0,  0.40}, -- bar tracks

        -- Text hierarchy (4 levels)
        textPrimary   = {1.0, 1.0, 1.0},
        textSecondary = {0.9, 0.9, 0.9},
        textMuted     = {0.6, 0.6, 0.6},
        textDisabled  = {0.4, 0.4, 0.4},

        -- Accent (specific roles: value, upgrade indicator)
        accentGold     = {1.0, 0.82, 0.0},
        accentGoldFill = {1.0, 0.82, 0.0, 0.70},

        -- Status colors
        statusEquipped    = {0.3, 1.0, 0.3},
        statusInBags      = {0.3, 0.6, 1.0},
        statusUpgradeable = {1.0, 0.82, 0.0},
        statusMissing     = {1.0, 0.3, 0.3},

        -- Source types (canonical, used by both PriorityTab and VaultTab)
        sourceDungeon = {0.4, 0.8, 1.0},
        sourceRaid    = {1.0, 0.5, 0.2},
        sourceWorld   = {0.6, 1.0, 0.4},
        sourceCrafted = {0.5, 1.0, 0.5},
        sourceDelve   = {0.8, 0.5, 1.0},
        sourcePvP     = {1.0, 0.4, 0.5},

        -- Subtle whites (dividers, faint tints)
        whiteFaint   = {1, 1, 1, 0.03},
        whiteSoft    = {1, 1, 1, 0.05},
        whiteDivider = {1, 1, 1, 0.08},

        -- Class color (resolved at runtime by Theme:Init())
        classAccent     = {1, 1, 1},
        classAccentSoft = {1, 1, 1, 0.15},
    },

    font = {
        title      = "GameFontHighlightLarge",   -- main frame title
        label      = "GameFontNormal",           -- primary row labels
        heading    = "GameFontHighlightSmall",   -- section headers (white)
        subheading = "GameFontHighlightSmall",   -- minor headers (white)
        body       = "GameFontNormalSmall",      -- default body (gold)
        bodyBright = "GameFontHighlightSmall",   -- emphasized body (white)
        emphasis   = "GameFontNormalLarge",      -- ranks/scores
    },

    space = {
        xs  = 4,
        sm  = 8,
        md  = 12,
        lg  = 16,
        xl  = 20,
        xxl = 24,
    },

    border = {
        frame = {
            edgeFile = "Interface\\Buttons\\WHITE8X8",
            edgeSize = 2,
            insets   = {0, 0, 0, 0},
        },
        row = {
            edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border",
            edgeSize = 12,
            insets   = {3, 3, 3, 3},
        },
        rowFlat = {
            insets = {3, 3, 3, 3},
        },
    },

    size = {
        rowHeightStandard = 22,
        rowHeightLarge    = 52,
        rowHeightSmall    = 20,
        rowHeightVault    = 24,
        barHeight         = 14,
        barHeightSmall    = 8,
        tabButtonW        = 80,
        tabButtonH        = 24,
        statusIcon        = 14,
    },
}

-- Resolve class color at runtime. Called from Core.lua after the player class is known.
function GearPath.Theme:Init()
    local _, classFile = UnitClass("player")
    if classFile and RAID_CLASS_COLORS[classFile] then
        local c = RAID_CLASS_COLORS[classFile]
        self.color.classAccent     = {c.r, c.g, c.b}
        self.color.classAccentSoft = {c.r, c.g, c.b, 0.15}
    end
end
