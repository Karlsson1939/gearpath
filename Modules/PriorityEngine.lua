-- GearPath
-- Modules/PriorityEngine.lua - Content source ranking algorithm

GearPath.PriorityEngine = {}
local PriorityEngine = GearPath.PriorityEngine

PriorityEngine.rankedSources = {}
PriorityEngine.missingItems  = {}

function PriorityEngine:Rebuild()
    self.rankedSources = {}
    self.missingItems  = {}

    local class = GearPath.currentClass
    local spec  = GearPath.currentSpec

    if not class or not spec then return end

    local bisSet = GearPath.BiSData
        and GearPath.BiSData[class]
        and GearPath.BiSData[class][spec]

    if not bisSet then return end

    local slotWeights = GearPath.db.profile.slotWeights
    local defaults    = GearPath.DefaultSlotWeights or {}
    local scanner     = GearPath.GearScanner
    local filters     = GearPath.db.profile.filters

    local sources = {}

    for slotID, bisItem in pairs(bisSet) do
        if self:IsSourceAllowed(bisItem.sourceType, filters) then
            local status = self:GetSlotStatus(slotID, bisItem, scanner)

            if status == "MISSING" or status == "UPGRADEABLE" then
                local weight = (slotWeights and slotWeights[slotID])
                    or defaults[slotID]
                    or 1.0

                local score = bisItem.priority * weight
                local sourceKey = bisItem.source

                if not sources[sourceKey] then
                    sources[sourceKey] = {
                        source     = sourceKey,
                        sourceName = bisItem.sourceName,
                        sourceType = bisItem.sourceType,
                        score      = 0,
                        items      = {},
                    }
                end

                sources[sourceKey].score = sources[sourceKey].score + score
                table.insert(sources[sourceKey].items, {
                    slotID  = slotID,
                    bisItem = bisItem,
                    status  = status,
                    score   = score,
                })

                table.insert(self.missingItems, {
                    slotID  = slotID,
                    bisItem = bisItem,
                    status  = status,
                })
            end
        end
    end

    for _, sourceData in pairs(sources) do
        table.sort(sourceData.items, function(a, b)
            return a.score > b.score
        end)
        table.insert(self.rankedSources, sourceData)
    end

    table.sort(self.rankedSources, function(a, b)
        return a.score > b.score
    end)
end

function PriorityEngine:GetSlotStatus(slotID, bisItem, scanner)
    if not scanner then return "MISSING" end

    local equipped = scanner.equipped[slotID]
    local bagged   = scanner.bagged

    -- No BiS item defined for this slot (itemID = 0 means unresolved)
    if not bisItem.itemID or bisItem.itemID == 0 then
        return "MISSING"
    end

    -- Exact BiS item equipped
    if equipped and equipped.itemID and equipped.itemID == bisItem.itemID then
        return "EQUIPPED"
    end

    -- BiS item sitting in bags
    if bagged and bagged[bisItem.itemID] then
        return "IN_BAGS"
    end

    -- Something else equipped here at higher ilvl — slot is an upgrade target,
    -- not completely missing, but still worth tracking in the priority list
    if equipped and equipped.itemID ~= 0 and
       equipped.ilvl and equipped.ilvl > 0 and equipped.ilvl > bisItem.ilvl then
        return "UPGRADEABLE"
    end

    return "MISSING"
end

function PriorityEngine:IsSourceAllowed(sourceType, filters)
    if not filters then return true end
    if sourceType == "DUNGEON" and not filters.showDungeons then return false end
    if sourceType == "RAID"    and not filters.showRaid     then return false end
    if sourceType == "CRAFTED" and not filters.showCrafted  then return false end
    if sourceType == "WORLD"   and not filters.showWorld    then return false end
    if sourceType == "DELVE"   and not filters.showDelves   then return false end
    if sourceType == "PVP"     and not filters.showPvP      then return false end
    return true
end

function PriorityEngine:GetTop(n)
    local result = {}
    for i = 1, math.min(n, #self.rankedSources) do
        local s = self.rankedSources[i]
        result[i] = {
            name         = s.sourceName,
            score        = s.score,
            missingCount = #s.items,
            items        = s.items,
        }
    end
    return result
end

function PriorityEngine:PrintTop(n)
    if #self.rankedSources == 0 then
        GearPath:Print("No missing BiS items found. You're fully geared!")
        return
    end

    GearPath:Print(string.format("=== Top %d Priority Sources ===", n))
    for i = 1, math.min(n, #self.rankedSources) do
        local s = self.rankedSources[i]
        GearPath:Print(string.format(
            "  %d. %s [%s] — Score: %.1f — %d item(s) missing",
            i, s.sourceName, s.sourceType, s.score, #s.items
        ))
        for _, item in ipairs(s.items) do
            GearPath:Print(string.format(
                "      - %s (%s)",
                item.bisItem.itemName,
                item.status
            ))
        end
    end
end

function PriorityEngine:GetMissingCount()
    return #self.missingItems
end