-- GearPath
-- UI/BiSTab.lua - BiS checklist view

GearPath.BiSTab = {}
local BiSTab = GearPath.BiSTab

local container = nil

local STATUS_ICONS = {
    EQUIPPED    = { text = "✓", r = 0.3, g = 1.0, b = 0.3 },
    IN_BAGS     = { text = "B", r = 0.3, g = 0.6, b = 1.0 },
    UPGRADEABLE = { text = "↑", r = 1.0, g = 0.82, b = 0.0 },
    MISSING     = { text = "✗", r = 1.0, g = 0.3, b = 0.3 },
}

function BiSTab:Show(parent)
    if not container then
        container = CreateFrame("ScrollFrame", "GearPathBiSTab", parent, "UIPanelScrollFrameTemplate")
        container:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, 0)
        container:SetPoint("BOTTOMRIGHT", parent, "BOTTOMRIGHT", -20, 0)

        container.child = CreateFrame("Frame", nil, container)
        container.child:SetSize(parent:GetWidth() - 20, 600)
        container:SetScrollChild(container.child)
    else
        container:SetParent(parent)
        container:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, 0)
        container:SetPoint("BOTTOMRIGHT", parent, "BOTTOMRIGHT", -20, 0)
    end

    container:Show()
    self:Refresh()
end

function BiSTab:Hide()
    if container then container:Hide() end
end

function BiSTab:Refresh()
    if not container or not container:IsShown() then return end

    -- Clear child
    local child = container.child
    for _, child2 in pairs({ child:GetChildren() }) do
        child2:Hide()
    end

    local bisSet = GearPath:GetBiSForCurrentSpec()
    if not bisSet then return end

    local scanner = GearPath.GearScanner
    local engine  = GearPath.PriorityEngine
    local yOffset = -4
    local slotOrder = { 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17 }

    for _, slotID in ipairs(slotOrder) do
        local bisItem = bisSet[slotID]
        if bisItem then
            local status = engine and engine:GetSlotStatus(slotID, bisItem, scanner) or "MISSING"
            local icon   = STATUS_ICONS[status] or STATUS_ICONS["MISSING"]

            local row = CreateFrame("Frame", nil, child)
            row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, yOffset)
            row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, yOffset)
            row:SetHeight(22)

            -- Alternating row bg
            if (yOffset / -22) % 2 == 0 then
                local bg = row:CreateTexture(nil, "BACKGROUND")
                bg:SetAllPoints()
                bg:SetColorTexture(1, 1, 1, 0.03)
            end

            -- Status icon
            local statusText = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            statusText:SetPoint("LEFT", row, "LEFT", 4, 0)
            statusText:SetText(icon.text)
            statusText:SetTextColor(icon.r, icon.g, icon.b)
            statusText:SetWidth(14)

            -- Slot name
            local slotLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            slotLabel:SetPoint("LEFT", statusText, "RIGHT", 4, 0)
            slotLabel:SetText(self:GetSlotName(slotID))
            slotLabel:SetTextColor(0.6, 0.6, 0.6)
            slotLabel:SetWidth(70)

            -- Item name
            local itemLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            itemLabel:SetPoint("LEFT", slotLabel, "RIGHT", 4, 0)
            itemLabel:SetText(bisItem.itemName)
            itemLabel:SetTextColor(1.0, 0.82, 0.0)

            -- Source
            local sourceLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            sourceLabel:SetPoint("RIGHT", row, "RIGHT", -4, 0)
            sourceLabel:SetText(bisItem.sourceName)
            sourceLabel:SetTextColor(0.5, 0.5, 0.5)

            yOffset = yOffset - 22
        end
    end

    child:SetHeight(math.abs(yOffset) + 10)
end

function BiSTab:GetSlotName(slotID)
    local names = {
        [1]  = "Head",     [2]  = "Neck",     [3]  = "Shoulders",
        [5]  = "Chest",    [6]  = "Waist",    [7]  = "Legs",
        [8]  = "Feet",     [9]  = "Wrists",   [10] = "Hands",
        [11] = "Finger 1", [12] = "Finger 2", [13] = "Trinket 1",
        [14] = "Trinket 2",[15] = "Back",     [16] = "Main Hand",
        [17] = "Off Hand",
    }
    return names[slotID] or "Unknown"
end