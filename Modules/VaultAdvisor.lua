-- GearPath
-- Modules/VaultAdvisor.lua - Great Vault slot analysis and BiS recommendations

GearPath.VaultAdvisor = {}
local VaultAdvisor = GearPath.VaultAdvisor

-- Threshold counts per category to unlock each of the 3 vault slots
-- M+: 1 / 4 / 8 runs | Raid: 1 / 2 / 4 bosses | World/Delve: 1 / 3 / 5
local THRESHOLDS = {
    [1] = { 1, 4, 8  },  -- Activities (Mythic+)
    [2] = { 1, 2, 4  },  -- Raid
    [3] = { 1, 3, 5  },  -- World / Delve
}

local TYPE_NAMES = {
    [1] = "Mythic+",
    [2] = "Raid",
    [3] = "World / Delve",
}

-- Parsed vault state, rebuilt on Refresh()
VaultAdvisor.slots     = {}  -- list of { type, typeLabel, slotIndex, unlocked, itemLevel, progress, threshold }
VaultAdvisor.available = false

function VaultAdvisor:Refresh()
    self.slots     = {}
    self.available = false

    if not C_WeeklyRewards then return end

    local activities = C_WeeklyRewards.GetActivities()
    if not activities or #activities == 0 then return end

    self.available = true

    -- Group activities by type, sorted by threshold ascending
    local byType = {}
    for _, act in ipairs(activities) do
        local t = act.type
        if not byType[t] then byType[t] = {} end
        table.insert(byType[t], act)
    end

    for typeID, acts in pairs(byType) do
        -- Sort by threshold so slot 1 < slot 2 < slot 3
        table.sort(acts, function(a, b)
            return a.threshold < b.threshold
        end)

        for slotIndex, act in ipairs(acts) do
            local unlocked = act.progress >= act.threshold
            table.insert(self.slots, {
                type      = typeID,
                typeLabel = TYPE_NAMES[typeID] or "Unknown",
                slotIndex = slotIndex,
                unlocked  = unlocked,
                itemLevel = act.itemLevel or 0,
                progress  = act.progress  or 0,
                threshold = act.threshold or 0,
                claimID   = act.claimID,
            })
        end
    end

    -- Sort: unlocked first, then by ilvl descending
    table.sort(self.slots, function(a, b)
        if a.unlocked ~= b.unlocked then
            return a.unlocked and not b.unlocked
        end
        return a.itemLevel > b.itemLevel
    end)
end

-- Returns which vault slot is most valuable to fill based on BiS list.
-- For each unlocked vault slot, checks if any BiS item could come from that
-- activity type and scores it. Returns ranked list for the UI to display.
function VaultAdvisor:GetRankedRecommendations()
    local result = {}

    local bisSet = GearPath.BiSData
        and GearPath.BiSData[GearPath.currentClass]
        and GearPath.BiSData[GearPath.currentClass][GearPath.currentSpec]

    local scanner = GearPath.GearScanner
    local engine  = GearPath.PriorityEngine

    for _, slot in ipairs(self.slots) do
        local rec = {
            slot       = slot,
            bisMatches = {},
            score      = 0,
        }

        if bisSet and scanner and engine then
            for slotID, bisItem in pairs(bisSet) do
                local status = engine:GetSlotStatus(slotID, bisItem, scanner)
                if status == "MISSING" or status == "UPGRADEABLE" then
                    -- Check if this BiS item's source type matches the vault category
                    local matches = self:SourceMatchesVaultType(bisItem.sourceType, slot.type)
                    if matches then
                        table.insert(rec.bisMatches, {
                            slotID   = slotID,
                            bisItem  = bisItem,
                            status   = status,
                            priority = bisItem.priority or 1,
                        })
                        rec.score = rec.score + (bisItem.priority or 1)
                    end
                end
            end
        end

        table.sort(rec.bisMatches, function(a, b)
            return a.priority > b.priority
        end)

        table.insert(result, rec)
    end

    -- Sort recommendations: unlocked + high score first
    table.sort(result, function(a, b)
        local aU = a.slot.unlocked and 1 or 0
        local bU = b.slot.unlocked and 1 or 0
        if aU ~= bU then return aU > bU end
        return a.score > b.score
    end)

    return result
end

-- Maps a BiS item's sourceType to the vault activity category it could appear in
function VaultAdvisor:SourceMatchesVaultType(sourceType, vaultType)
    if vaultType == 1 then  -- Mythic+
        return sourceType == "DUNGEON"
    elseif vaultType == 2 then  -- Raid
        return sourceType == "RAID"
    elseif vaultType == 3 then  -- World / Delve
        return sourceType == "WORLD" or sourceType == "DELVE"
    end
    return false
end

function VaultAdvisor:PrintSummary()
    if not self.available then
        GearPath:Print("Vault data not available. Open the Great Vault UI first.")
        return
    end

    GearPath:Print("=== Great Vault ===")
    for _, slot in ipairs(self.slots) do
        local status = slot.unlocked and "UNLOCKED" or string.format("%d/%d", slot.progress, slot.threshold)
        GearPath:Print(string.format("  %s Slot %d — ilvl %d — %s",
            slot.typeLabel, slot.slotIndex, slot.itemLevel, status))
    end

    local recs = self:GetRankedRecommendations()
    GearPath:Print("=== Vault Recommendations ===")
    for i, rec in ipairs(recs) do
        if rec.slot.unlocked and #rec.bisMatches > 0 then
            GearPath:Print(string.format("  %d. %s Slot %d (ilvl %d) — %d BiS match(es)",
                i, rec.slot.typeLabel, rec.slot.slotIndex,
                rec.slot.itemLevel, #rec.bisMatches))
            for j, match in ipairs(rec.bisMatches) do
                if j <= 3 then
                    GearPath:Print(string.format("      - %s [%s]",
                        match.bisItem.itemName, match.status))
                end
            end
        end
    end
end