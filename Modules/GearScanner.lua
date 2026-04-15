-- GearPath
-- Modules/GearScanner.lua - Equipment and bag scanning

GearPath.GearScanner = {}
local GearScanner = GearPath.GearScanner

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

-- Equipment slot type IDs that we care about
-- These match the invType returned by GetItemInfo()
local EQUIPPABLE_INV_TYPES = {
    INVTYPE_HEAD = true,
    INVTYPE_NECK = true,
    INVTYPE_SHOULDER = true,
    INVTYPE_CHEST = true,
    INVTYPE_ROBE = true,
    INVTYPE_WAIST = true,
    INVTYPE_LEGS = true,
    INVTYPE_FEET = true,
    INVTYPE_WRIST = true,
    INVTYPE_HAND = true,
    INVTYPE_FINGER = true,
    INVTYPE_TRINKET = true,
    INVTYPE_CLOAK = true,
    INVTYPE_WEAPON = true,
    INVTYPE_SHIELD = true,
    INVTYPE_2HWEAPON = true,
    INVTYPE_WEAPONMAINHAND = true,
    INVTYPE_WEAPONOFFHAND = true,
    INVTYPE_HOLDABLE = true,
    INVTYPE_RANGED = true,
    INVTYPE_RANGEDRIGHT = true,
}

GearScanner.equipped = {}
GearScanner.bagged   = {}

function GearScanner:Scan(class, spec)
    self.equipped = {}
    self.bagged   = {}
    self:ScanEquipped()
    self:ScanBags()
end

function GearScanner:GetIlvlFromLink(itemLink)
    if not itemLink then return 0 end
    -- Item links encode the effective ilvl in the 9th field of the item string
    -- Format: item:itemID:enchant:gem1:gem2:gem3:gem4:suffixID:uniqueID:linkLevel:specializationID:upgradeTypeID:instanceDifficultyID:numBonusIDs:bonusID1:...:upgradeValue
    local _, _, _, _, _, _, _, _, _, _, _, _, _, ilvl = GetItemInfo(itemLink)
    if ilvl and ilvl > 0 then return ilvl end
    -- Fallback: parse ilvl from the link string directly
    local linkIlvl = tonumber(itemLink:match("item:%d+:%d+:%d+:%d+:%d+:%d+:%d+:%d+:(%d+)"))
    return linkIlvl or 0
end

function GearScanner:ScanEquipped()
    for slotID, slotName in pairs(SLOT_NAMES) do
        local itemID   = GetInventoryItemID("player", slotID)
        local itemLink = GetInventoryItemLink("player", slotID)
        local ilvl     = 0

        if itemLink then
            local _, _, itemRarity, itemLevel = GetItemInfo(itemLink)
            -- GetItemInfo returns the effective ilvl as the 4th return value
            ilvl = itemLevel or 0
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
                if itemLink then
                    local _, _, _, _, _, _, _, _, invType = GetItemInfo(itemLink)
                    -- Only track equippable gear, ignore consumables/reagents/etc
                    if invType and EQUIPPABLE_INV_TYPES[invType] then
                        local _, _, _, ilvl = GetItemInfo(itemLink)
                        self.bagged[itemID] = {
                            itemID    = itemID,
                            itemLink  = itemLink,
                            ilvl      = ilvl or 0,
                            bagIndex  = bagIndex,
                            slotIndex = slotIndex,
                        }
                    end
                end
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
    GearPath:Print(string.format("=== Bags: %d equippable items tracked ===", baggedCount))
end