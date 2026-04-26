-- GearPath
-- Modules/Detection.lua - Class, specialization, and hero talent auto-detection

GearPath.Detection = {}
local Detection = GearPath.Detection

local classDisplayNames = {
    WARRIOR     = "Warrior",
    PALADIN     = "Paladin",
    HUNTER      = "Hunter",
    ROGUE       = "Rogue",
    PRIEST      = "Priest",
    DEATHKNIGHT = "Death Knight",
    SHAMAN      = "Shaman",
    MAGE        = "Mage",
    WARLOCK     = "Warlock",
    MONK        = "Monk",
    DRUID       = "Druid",
    DEMONHUNTER = "Demon Hunter",
    EVOKER      = "Evoker",
}

-- Normalize a name to a BiSData key by stripping whitespace.
-- API returns "Beast Mastery" / "Pack Leader", our keys are "BeastMastery" / "PackLeader".
local function normalizeKey(name)
    if not name then return nil end
    return (name:gsub("%s+", ""))
end

function Detection:Detect(callback)
    local _, classFile = UnitClass("player")

    if not classFile then
        return
    end

    local specIndex = GetSpecialization()

    if not specIndex then
        C_Timer.After(1, function()
            Detection:Detect(callback)
        end)
        return
    end

    local specID, specName = GetSpecializationInfo(specIndex)

    if not specID or not specName then
        C_Timer.After(1, function()
            Detection:Detect(callback)
        end)
        return
    end

    local normalizedSpec = normalizeKey(specName)

    -- Hero talent detection
    local heroTalentDisplay = nil
    local normalizedHeroTalent = nil
    local subTreeID = C_ClassTalents.GetActiveHeroTalentSpec()

    if subTreeID then
        local configID = C_ClassTalents.GetActiveConfigID()
        if not configID then
            -- Talent config not yet loaded at login; retry like we do for spec.
            C_Timer.After(1, function()
                Detection:Detect(callback)
            end)
            return
        end

        local subTreeInfo = C_Traits.GetSubTreeInfo(configID, subTreeID)
        if subTreeInfo and subTreeInfo.name then
            heroTalentDisplay = subTreeInfo.name
            normalizedHeroTalent = normalizeKey(heroTalentDisplay)
        end
    end

    -- Store on the main addon table. We keep both forms:
    --   currentSpec / currentHeroTalent  = normalized keys for BiSData lookup
    --   currentSpecDisplay / ...Display  = human-readable for UI and chat
    GearPath.currentClass             = classFile
    GearPath.currentSpec              = normalizedSpec
    GearPath.currentSpecDisplay       = specName
    GearPath.currentHeroTalent        = normalizedHeroTalent
    GearPath.currentHeroTalentDisplay = heroTalentDisplay

    if callback then
        callback(classFile, normalizedSpec, normalizedHeroTalent)
    end
end

function Detection:GetSummary()
    if not GearPath.currentClass or not GearPath.currentSpecDisplay then
        return "Unknown"
    end
    local base = GearPath.currentSpecDisplay .. " " .. (classDisplayNames[GearPath.currentClass] or GearPath.currentClass)
    if GearPath.currentHeroTalentDisplay then
        return base .. " (" .. GearPath.currentHeroTalentDisplay .. ")"
    end
    return base
end

function Detection:IsReady()
    -- "Ready" = max level + full identity (class + spec + hero talent).
    -- GearPath is endgame-oriented; without all three, we don't show BiS data.
    local maxLevel = GetMaxLevelForPlayerExpansion and GetMaxLevelForPlayerExpansion() or 80
    if UnitLevel("player") < maxLevel then
        return false, "not_max_level"
    end
    if not GearPath.currentClass or not GearPath.currentSpec then
        return false, "no_spec"
    end
    if not GearPath.currentHeroTalent then
        return false, "no_hero_talent"
    end
    return true, nil
end

function Detection:GetNotReadyMessage()
    local ready, reason = Detection:IsReady()
    if ready then return nil end

    if reason == "not_max_level" then
        return "GearPath is an endgame addon. Reach max level to see your BiS priorities."
    elseif reason == "no_hero_talent" then
        return "Select a hero talent tree in your talent window to see your BiS priorities."
    else
        return "GearPath is still detecting your character. If this persists, try /reload."
    end
end