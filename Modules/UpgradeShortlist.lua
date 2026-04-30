-- GearPath
-- Modules/UpgradeShortlist.lua - Top-10 missing BiS shortlist for Vault picks

GearPath.UpgradeShortlist = {}
local UpgradeShortlist = GearPath.UpgradeShortlist

UpgradeShortlist.topItems = {}

function UpgradeShortlist:Build()
    self.topItems = {}

    local bisSet  = GearPath:GetBiSForCurrentSpec()
    if not bisSet then return end

    local scanner = GearPath.GearScanner
    local engine  = GearPath.PriorityEngine
    local filters = GearPath.db.profile.filters
    if not scanner or not engine then return end

    local candidates = {}

    for slotID, bisItem in pairs(bisSet) do
        if engine:IsSourceAllowed(bisItem.sourceType, filters) then
            local status = engine:GetSlotStatus(slotID, bisItem, scanner)

            if status == "MISSING" or status == "UPGRADEABLE" then
                local score    = engine:ScoreItem(slotID, bisItem)
                local slotName = scanner.SLOT_NAMES[slotID] or ("Slot " .. slotID)

                table.insert(candidates, {
                    slotID   = slotID,
                    slotName = slotName,
                    bisItem  = bisItem,
                    status   = status,
                    score    = score,
                })
            end
        end
    end

    -- Sort by score descending, ties broken by slotID ascending
    table.sort(candidates, function(a, b)
        if a.score ~= b.score then
            return a.score > b.score
        end
        return a.slotID < b.slotID
    end)

    -- Keep top 10
    for i = 1, math.min(10, #candidates) do
        self.topItems[i] = candidates[i]
    end
end

function UpgradeShortlist:PrintSummary()
    if #self.topItems == 0 then
        GearPath:Print("You're fully BiS for this spec.")
        return
    end

    GearPath:Print("=== Top Upgrade Gaps ===")
    for i, entry in ipairs(self.topItems) do
        if i > 5 then break end
        GearPath:Print(string.format("  %d. %s — %s [%s] — %.1f",
            i, entry.slotName, entry.bisItem.itemName,
            entry.status, entry.score))
    end
end
