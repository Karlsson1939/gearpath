-- GearPath
-- Core.lua - Addon initialization, namespace, event registration

-- Create the addon namespace
GearPath = LibStub("AceAddon-3.0"):NewAddon("GearPath", "AceEvent-3.0", "AceConsole-3.0")

-- Default saved variable schema
local defaults = {
    profile = {
        minimap = {
            hide = false,
            minimapPos = 220,
        },
        ui = {
            scale = 1.0,
            fontSize = 12,
            activeTab = 1,
            framePoint = { point = "CENTER", x = 0, y = 0 },
            frameWidth = 420,
            frameHeight = 520,
        },
        slotWeights = {},
        filters = {
            showDungeons = true,
            showRaid = true,
            showCrafted = true,
            showWorld = true,
            showDelves = true,
            showPvP = false,
        },
    },
    char = {
        lastSpec = nil,
        lastHeroTalent = nil,
        manualOverrides = {},
    },
}

function GearPath:OnInitialize()
    self.db = LibStub("AceDB-3.0"):New("GearPathDB", defaults, true)

    self:RegisterChatCommand("gp", "SlashCommand")
    self:RegisterChatCommand("gearpath", "SlashCommand")

    self:Print("GearPath loaded. Type /gp to open.")
end

function GearPath:OnEnable()
    self:RegisterEvent("PLAYER_ENTERING_WORLD", "OnPlayerEnteringWorld")
    self:RegisterEvent("ACTIVE_TALENT_GROUP_CHANGED", "OnSpecChanged")
    self:RegisterEvent("PLAYER_SPECIALIZATION_CHANGED", "OnSpecChanged")
    self:RegisterEvent("TRAIT_CONFIG_UPDATED", "OnTraitConfigUpdated")
    self:RegisterEvent("PLAYER_LEVEL_UP", "OnPlayerLevelUp")
    self:RegisterEvent("PLAYER_EQUIPMENT_CHANGED", "OnEquipmentChanged")
    self:RegisterEvent("BAG_UPDATE", "OnBagUpdate")
    self:RegisterEvent("WEEKLY_REWARDS_UPDATE", "OnVaultUpdate")
    -- Re-scan when item cache populates (fires after uncached GetItemInfo resolves)
    self:RegisterEvent("GET_ITEM_INFO_RECEIVED", "OnItemInfoReceived")

    if GearPath.MinimapButton then
        GearPath.MinimapButton:Initialize()
    end
end

function GearPath:OnDisable()
    self:UnregisterAllEvents()
end

-- ============================================================
-- Event Handlers
-- ============================================================

function GearPath:OnPlayerEnteringWorld()
    if GearPath.Theme then
        GearPath.Theme:Init()
    end
    self:DetectAndLoad()
end

function GearPath:OnSpecChanged()
    -- Spec swap also changes the available hero talent trees, so re-detect fully.
    self:DetectAndLoad()
end

function GearPath:OnTraitConfigUpdated()
    -- Fires whenever the player changes talents (including hero talent selection).
    -- Re-detect so currentHeroTalent stays in sync without requiring a /reload.
    self:DetectAndLoad()
end

function GearPath:OnPlayerLevelUp()
    -- When the player dings max level, the "not ready" gate should resolve
    -- on its own. Re-detect so downstream modules refresh.
    self:DetectAndLoad()
end

local gearUpdateTimer = nil

function GearPath:OnEquipmentChanged()
    self:ScheduleGearUpdate()
end

function GearPath:OnBagUpdate()
    self:ScheduleGearUpdate()
end

function GearPath:ScheduleGearUpdate()
    if gearUpdateTimer then
        gearUpdateTimer:Cancel()
    end
    gearUpdateTimer = C_Timer.NewTimer(0.5, function()
        gearUpdateTimer = nil
        GearPath:RebuildPriority()
    end)
end

function GearPath:OnVaultUpdate()
    if GearPath.VaultAdvisor then
        GearPath.VaultAdvisor:Refresh()
    end
    if GearPath.VaultTab then
        GearPath.VaultTab:Refresh()
    end
end

local itemInfoPending = false
function GearPath:OnItemInfoReceived()
    -- Debounce: only rebuild once after a burst of item cache responses at login
    if itemInfoPending then return end
    itemInfoPending = true
    C_Timer.After(0.5, function()
        itemInfoPending = false
        GearPath:RebuildPriority()
    end)
end

-- ============================================================
-- Core Flow
-- ============================================================

function GearPath:DetectAndLoad()
    if GearPath.Detection then
        GearPath.Detection:Detect(function(class, spec, heroTalent)
            GearPath:OnDetectionComplete(class, spec, heroTalent)
        end)
    end
end

-- ============================================================
-- BiS Data Accessor
-- ============================================================

-- GetBiSForCurrentSpec returns the slot→item table that's correct for the
-- player's current class, spec, and hero talent — or nil if any of those
-- aren't set, or if we don't have data for this combination.
--
-- Schema: BiSData[class][spec][heroKey] → { [slotID] = bisItem, ... }
--
-- Resolution order:
--   1. Exact match on currentHeroTalent (e.g. "San'layn")
--   2. Fallback to "any" (the common case — 38/40 specs)
--   3. nil if neither exists
--
-- All readers that previously did BiSData[class][spec] should call this
-- instead, so the "which hero talent's list?" logic lives in one place.
function GearPath:GetBiSForCurrentSpec()
    if not self.BiSData then return nil end

    local class = self.currentClass
    local spec  = self.currentSpec
    if not class or not spec then return nil end

    local specData = self.BiSData[class] and self.BiSData[class][spec]
    if not specData then return nil end

    -- Per-hero-talent list takes precedence
    local hero = self.currentHeroTalent
    if hero and specData[hero] then
        return specData[hero]
    end

    -- Common case: single "any" list covers all hero talents
    return specData["any"]
end

-- Schema: GuideData[class][spec][heroKey] → { Stats, Gems, Enchants, Consumables }
-- Same resolution as GetBiSForCurrentSpec: hero-specific → "any" fallback → nil.
function GearPath:GetGuideForCurrentSpec()
    if not self.GuideData then return nil end

    local class = self.currentClass
    local spec  = self.currentSpec
    if not class or not spec then return nil end

    local specData = self.GuideData[class] and self.GuideData[class][spec]
    if not specData then return nil end

    local hero = self.currentHeroTalent
    if hero and specData[hero] then
        return specData[hero]
    end

    return specData["any"]
end

function GearPath:OnDetectionComplete(class, spec, heroTalent)
    self.currentClass       = class
    self.currentSpec        = spec
    self.currentHeroTalent  = heroTalent
    self.db.char.lastSpec        = spec
    self.db.char.lastHeroTalent  = heroTalent

    self:RebuildPriority()
end

function GearPath:RebuildPriority()
    -- Endgame-only gate: BiS data is only meaningful when we know the full
    -- identity (class + spec + hero talent) and the player is at max level.
    if not GearPath.Detection then return end

    local ready = GearPath.Detection:IsReady()
    if not ready then
        -- Keep saved scanner/priority state cleared so stale data doesn't show.
        if GearPath.MainFrame and GearPath.MainFrame:IsShown() then
            GearPath.MainFrame:Refresh()
        end
        return
    end

    if GearPath.GearScanner then
        GearPath.GearScanner:Scan(self.currentClass, self.currentSpec, self.currentHeroTalent)
    end

    if GearPath.PriorityEngine then
        GearPath.PriorityEngine:Rebuild()
    end

    if GearPath.MainFrame and GearPath.MainFrame:IsShown() then
        GearPath.MainFrame:Refresh()
    end
end

-- ============================================================
-- Slash Commands
-- ============================================================

function GearPath:SlashCommand(input)
    input = input:trim():lower()

    if input == "" then
        if GearPath.MainFrame then
            GearPath.MainFrame:Toggle()
        end
    elseif input == "priority" then
        if GearPath.PriorityEngine then
            GearPath.PriorityEngine:PrintTop(3)
        end
    elseif input == "bis" then
        if GearPath.GearScanner then
            GearPath.GearScanner:PrintSummary()
        end
    elseif input == "vault" then
        if GearPath.VaultAdvisor then
            GearPath.VaultAdvisor:PrintSummary()
        end
    elseif input == "reset" then
        if GearPath.MainFrame then
            GearPath.MainFrame:ResetPosition()
        end
        self:Print("Panel position reset.")
    elseif input == "status" then
        -- Quick debug: what does GearPath think you are right now?
        if GearPath.Detection then
            self:Print("Identity: " .. GearPath.Detection:GetSummary())
            local ready, reason = GearPath.Detection:IsReady()
            if ready then
                self:Print("Status: ready")
            else
                self:Print("Status: not ready (" .. (reason or "unknown") .. ")")
            end
        end
    else
        self:Print("GearPath commands:")
        self:Print("  /gp           Toggle the main panel")
        self:Print("  /gp priority  Print top priority sources to chat")
        self:Print("  /gp bis       Print equipped gear summary to chat")
        self:Print("  /gp vault     Print vault summary to chat")
        self:Print("  /gp status    Show current detection state")
        self:Print("  /gp reset     Reset panel position")
    end
end