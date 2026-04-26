-- GearPath
-- UI/BiSTab.lua - BiS checklist view

GearPath.BiSTab = {}
local BiSTab = GearPath.BiSTab

local container = nil

local STATUS_ICONS = {
    EQUIPPED    = { text = "✓", color = GearPath.Theme.color.statusEquipped },
    IN_BAGS     = { text = "B", color = GearPath.Theme.color.statusInBags },
    UPGRADEABLE = { text = "↑", color = GearPath.Theme.color.statusUpgradeable },
    MISSING     = { text = "✗", color = GearPath.Theme.color.statusMissing },
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

    local T = GearPath.Theme

    -- Clear child
    local child = container.child
    for _, child2 in pairs({ child:GetChildren() }) do
        child2:Hide()
    end
    for _, r in pairs({ child:GetRegions() }) do
        r:Hide()
    end

    local bisSet = GearPath:GetBiSForCurrentSpec()
    if not bisSet then
        local msg = child:CreateFontString(nil, "OVERLAY", T.font.label)
        msg:SetPoint("TOP", child, "TOP", 0, -40)
        msg:SetWidth(child:GetWidth() - 20)
        msg:SetText("No BiS data available for this spec.\nSelect a hero talent in-game to see your BiS list.")
        msg:SetTextColor(unpack(T.color.textMuted))
        msg:SetJustifyH("CENTER")
        child:SetHeight(100)
        return
    end

    local scanner = GearPath.GearScanner
    local engine  = GearPath.PriorityEngine
    local yOffset = -T.space.xs

    -- Status legend
    local legendX = T.space.xs
    local legendItems = {
        { text = "✓", color = T.color.statusEquipped, label = "Equipped" },
        { text = "B", color = T.color.statusInBags, label = "In Bags" },
        { text = "↑", color = T.color.statusUpgradeable, label = "Upgrade" },
        { text = "✗", color = T.color.statusMissing, label = "Missing" },
    }
    for _, item in ipairs(legendItems) do
        local icon = child:CreateFontString(nil, "OVERLAY", T.font.body)
        icon:SetPoint("TOPLEFT", child, "TOPLEFT", legendX, yOffset)
        icon:SetText(item.text)
        icon:SetTextColor(unpack(item.color))
        legendX = legendX + 12

        local lbl = child:CreateFontString(nil, "OVERLAY", T.font.body)
        lbl:SetPoint("TOPLEFT", child, "TOPLEFT", legendX, yOffset)
        lbl:SetText(item.label)
        lbl:SetTextColor(unpack(T.color.textMuted))
        legendX = legendX + lbl:GetStringWidth() + T.space.md
    end
    yOffset = yOffset - 16

    local slotOrder = { 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17 }

    for _, slotID in ipairs(slotOrder) do
        local bisItem = bisSet[slotID]
        if bisItem then
            local status = engine and engine:GetSlotStatus(slotID, bisItem, scanner) or "MISSING"
            local icon   = STATUS_ICONS[status] or STATUS_ICONS["MISSING"]

            local row = CreateFrame("Frame", nil, child)
            row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, yOffset)
            row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, yOffset)
            row:SetHeight(T.size.rowHeightStandard)

            -- Alternating row bg
            if (yOffset / -T.size.rowHeightStandard) % 2 == 0 then
                local bg = row:CreateTexture(nil, "BACKGROUND")
                bg:SetAllPoints()
                bg:SetColorTexture(unpack(T.color.whiteFaint))
            end

            -- Status icon
            local statusText = row:CreateFontString(nil, "OVERLAY", T.font.body)
            statusText:SetPoint("LEFT", row, "LEFT", T.space.xs, 0)
            statusText:SetText(icon.text)
            statusText:SetTextColor(unpack(icon.color))
            statusText:SetWidth(T.size.statusIcon)

            -- Slot name
            local slotLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
            slotLabel:SetPoint("LEFT", statusText, "RIGHT", T.space.xs, 0)
            slotLabel:SetText(self:GetSlotName(slotID))
            slotLabel:SetTextColor(unpack(T.color.textMuted))
            slotLabel:SetWidth(70)

            -- Item name
            local itemLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
            itemLabel:SetPoint("LEFT", slotLabel, "RIGHT", T.space.xs, 0)
            itemLabel:SetText(bisItem.itemName)
            itemLabel:SetTextColor(unpack(T.color.accentGold))

            -- Source
            local sourceLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
            sourceLabel:SetPoint("RIGHT", row, "RIGHT", -T.space.xs, 0)
            sourceLabel:SetText(bisItem.sourceName)
            sourceLabel:SetTextColor(unpack(T.color.textMuted))

            yOffset = yOffset - T.size.rowHeightStandard
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
