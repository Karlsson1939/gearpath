-- GearPath
-- Modules/Detection.lua - Class and specialization auto-detection

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

function Detection:Detect(callback)
    local _, classFile = UnitClass("player")

    if not classFile then
        GearPath:Print("[Detection] Could not read player class.")
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

    -- Normalize spec name to match BiSData keys (API returns e.g. "Beast Mastery",
    -- our keys use "BeastMastery" — strip all spaces)
    local normalizedSpec = specName:gsub("%s+", "")

    GearPath:Print(string.format("[Detection] Detected: %s %s (key: %s)",
        specName, classDisplayNames[classFile] or classFile, normalizedSpec))

    if callback then
        callback(classFile, normalizedSpec)
    end
end

function Detection:GetSummary()
    if not GearPath.currentClass or not GearPath.currentSpec then
        return "Unknown"
    end
    return GearPath.currentSpec .. " " .. (classDisplayNames[GearPath.currentClass] or GearPath.currentClass)
end