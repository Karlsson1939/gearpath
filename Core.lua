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
        manualOverrides = {},
    },
}

function GearPath:OnInitialize()
    self.db = LibStub("AceDB-3.0"):New("GearPathDB", defaults, true)

    self:RegisterChatCommand("gp", "SlashCommand")
    self:RegisterChatCommand("gearpath", "SlashCommand")

    self:Print("GearPath loaded. Type /gp for options.")
end

function GearPath:OnEnable()
    self:RegisterEvent("PLAYER_ENTERING_WORLD", "OnPlayerEnteringWorld")
    self:RegisterEvent("ACTIVE_TALENT_GROUP_CHANGED", "OnSpecChanged")
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
    self:DetectAndLoad()
end

function GearPath:OnSpecChanged()
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
        GearPath.Detection:Detect(function(class, spec)
            GearPath:OnDetectionComplete(class, spec)
        end)
    end
end

function GearPath:OnDetectionComplete(class, spec)
    self.currentClass = class
    self.currentSpec = spec
    self.db.char.lastSpec = spec

    self:RebuildPriority()
end

function GearPath:RebuildPriority()
    if not self.currentClass or not self.currentSpec then return end

    if GearPath.GearScanner then
        GearPath.GearScanner:Scan(self.currentClass, self.currentSpec)
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
    elseif input == "config" then
        if GearPath.Config then
            GearPath.Config:Open()
        end
    elseif input == "reset" then
        if GearPath.MainFrame then
            GearPath.MainFrame:ResetPosition()
        end
        self:Print("Panel position reset.")
    else
        self:Print("Commands: /gp | /gp priority | /gp bis | /gp vault | /gp config | /gp reset")
    end
end