-- GearPath
-- Modules/GearScanner.lua - Equipment and bag scanning

GearPath.GearScanner = {}
local GearScanner = GearPath.GearScanner

-- Equipment slot IDs and their display names
local SLOT_NAMES = {
    [1]  = "Head",
    [2]  = "Neck",
    [3]  = "Shoulders",
    [4]  = "Chest",
    [5]  = "Waist",
    [6]  = "Legs",
    [7]  = "Feet",
    [8]  = "Wrists",
    [9]  = "Hands",
    [10] = "Finger 1",
    [11] = "Finger 2",
    [12] = "Trinket 1",
    [13] = "Trinket 2",
    [14] = "Back",
    [15] = "Main Hand",
    [16] = "Off Hand",
}

-- Scanned results, keyed by slotID
GearScanner.equipped = {}
GearScanner.bagged   = {}

function GearScanner:Scan(class, spec)
    self.equipped = {}
    self.bagged   = {}

    self:ScanEquipped()
    self:ScanBags()
end

function GearScanner:ScanEquipped()
    for slotID, slotName in pairs(SLOT_NAMES) do
        local itemID   = GetInventoryItemID("player", slotID)
        local itemLink = GetInventoryItemLink("player", slotID)
        local ilvl     = 0

        if itemLink then
            local _, _, effectiveIlvl = GetDetailedItemLevelInfo(itemLink)
            ilvl = effectiveIlvl or 0
        end

        self.equipped[slotID] = {
            slotName = slotName,
            itemID   = itemID,
            itemLink = itemLink,
            ilvl     = ilvl,
        }
    end
end

function GearScanner:ScanBags()
    for bagIndex = 0, 4 do
        local numSlots = C_Container.GetContainerNumSlots(bagIndex)
        for slotIndex = 1, numSlots do
            local itemID = C_Container.GetContainerItemID(bagIndex, slotIndex)
            if itemID then
                local itemLink = C_Container.GetContainerItemLink(bagIndex, slotIndex)
                self.bagged[itemID] = {
                    itemID   = itemID,
                    itemLink = itemLink,
                    bagIndex = bagIndex,
                    slotIndex = slotIndex,
                }
            end
        end
    end
end

function GearScanner:PrintSummary()
    GearPath:Print("=== Equipped Gear ===")
    for slotID = 1, 16 do
        local slot = self.equipped[slotID]
        if slot then
            if slot.itemLink then
                GearPath:Print(string.format("  [%s] %s (ilvl %d)", slot.slotName, slot.itemLink, slot.ilvl))
            else
                GearPath:Print(string.format("  [%s] empty", slot.slotName))
            end
        end
    end

    local baggedCount = 0
    for _ in pairs(self.bagged) do baggedCount = baggedCount + 1 end
    GearPath:Print(string.format("=== Bags: %d unique items tracked ===", baggedCount))
end