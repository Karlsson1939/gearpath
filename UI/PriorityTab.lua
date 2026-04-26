-- GearPath
-- UI/PriorityTab.lua - Dungeon priority list view

GearPath.PriorityTab = {}
local PriorityTab = GearPath.PriorityTab

local container = nil
local rows = {}

local SOURCE_TYPE_COLORS = {
    DUNGEON = { 0.4, 0.8, 1.0 },
    RAID    = { 1.0, 0.5, 0.2 },
    CRAFTED = { 0.5, 1.0, 0.5 },
    WORLD   = { 1.0, 0.9, 0.3 },
    DELVE   = { 0.8, 0.5, 1.0 },
    PVP     = { 1.0, 0.3, 0.3 },
}

function PriorityTab:Show(parent)
    if not container then
        container = CreateFrame("Frame", "GearPathPriorityTab", parent)
        container:SetAllPoints(parent)
    else
        container:SetParent(parent)
        container:SetAllPoints(parent)
    end
    container:Show()
    self:Refresh()
end

function PriorityTab:Hide()
    if container then container:Hide() end
end

function PriorityTab:Refresh()
    if not container or not container:IsShown() then return end

    -- Clear existing rows
    for _, row in ipairs(rows) do
        row:Hide()
    end
    rows = {}

    local engine = GearPath.PriorityEngine
    if not engine or #engine.rankedSources == 0 then
        self:ShowEmptyState()
        return
    end

    if container.emptyLabel then
        container.emptyLabel:Hide()
    end

    self:DrawProgressBar()

    local yOffset = -36
    for i, sourceData in ipairs(engine.rankedSources) do
        local row = self:CreateSourceRow(container, sourceData, i, yOffset)
        table.insert(rows, row)
        yOffset = yOffset - 58
    end
end

function PriorityTab:DrawProgressBar()
    if container.progressBg then
        container.progressBg:Show()
        container.progressFill:Show()
        container.progressLabel:Show()
        container.progressCount:Show()
    else
        local bg = container:CreateTexture(nil, "BACKGROUND")
        bg:SetPoint("TOPLEFT", container, "TOPLEFT", 0, -4)
        bg:SetPoint("TOPRIGHT", container, "TOPRIGHT", 0, -4)
        bg:SetHeight(14)
        bg:SetColorTexture(0, 0, 0, 0.4)
        container.progressBg = bg

        local fill = container:CreateTexture(nil, "ARTWORK")
        fill:SetPoint("TOPLEFT", bg, "TOPLEFT", 1, -1)
        fill:SetHeight(12)
        fill:SetColorTexture(1.0, 0.65, 0.0, 0.7)
        container.progressFill = fill

        local label = container:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        label:SetPoint("LEFT", bg, "LEFT", 4, 0)
        label:SetText("BiS completion")
        label:SetTextColor(0.9, 0.9, 0.9)
        container.progressLabel = label

        local count = container:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        count:SetPoint("RIGHT", bg, "RIGHT", -4, 0)
        container.progressCount = count
    end

    local total    = 0
    local equipped = 0
    local bisSet   = GearPath:GetBiSForCurrentSpec()

    if bisSet then
        for _ in pairs(bisSet) do total = total + 1 end
        local scanner = GearPath.GearScanner
        if scanner then
            for slotID, bisItem in pairs(bisSet) do
                local status = GearPath.PriorityEngine:GetSlotStatus(slotID, bisItem, scanner)
                if status == "EQUIPPED" or status == "IN_BAGS" then
                    equipped = equipped + 1
                end
            end
        end
    end

    local pct   = total > 0 and (equipped / total) or 0
    local width = container:GetWidth() - 2
    container.progressFill:SetWidth(math.max(1, width * pct))
    container.progressCount:SetText(string.format("%d / %d slots", equipped, total))
    container.progressCount:SetTextColor(1.0, 0.82, 0.0)
end

function PriorityTab:RepositionRows()
    local ROW_GAP = 6
    local yOffset = -36
    for _, r in ipairs(rows) do
        r:ClearAllPoints()
        r:SetPoint("TOPLEFT", container, "TOPLEFT", 0, yOffset)
        r:SetPoint("TOPRIGHT", container, "TOPRIGHT", 0, yOffset)
        yOffset = yOffset - r:GetHeight() - ROW_GAP
    end
end

function PriorityTab:CreateSourceRow(parent, sourceData, index, yOffset)
    local ROW_HEIGHT  = 52
    local ITEM_HEIGHT = 20
    local expandedItems = {}
    local expanded    = false

    local row = CreateFrame("Button", nil, parent, "BackdropTemplate")
    row:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, yOffset)
    row:SetPoint("TOPRIGHT", parent, "TOPRIGHT", 0, yOffset)
    row:SetHeight(ROW_HEIGHT)
    row:SetBackdrop({
        bgFile   = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border",
        tile     = true, tileSize = 16, edgeSize = 12,
        insets   = { left = 3, right = 3, top = 3, bottom = 3 },
    })
    row:SetBackdropColor(0.08, 0.08, 0.12, 0.9)

    local color = SOURCE_TYPE_COLORS[sourceData.sourceType] or { 1, 1, 1 }

    -- Rank number
    local rank = row:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    rank:SetPoint("LEFT", row, "LEFT", 8, 4)
    rank:SetText(index)
    rank:SetTextColor(0.5, 0.5, 0.5)
    rank:SetWidth(20)

    -- Color tag
    local tag = row:CreateTexture(nil, "ARTWORK")
    tag:SetPoint("LEFT", rank, "RIGHT", 6, 0)
    tag:SetSize(3, 32)
    tag:SetColorTexture(color[1], color[2], color[3], 0.9)

    -- Source name
    local name = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    name:SetPoint("TOPLEFT", tag, "TOPRIGHT", 8, -4)
    name:SetText(sourceData.sourceName)
    name:SetTextColor(1, 1, 1)

    -- Type badge
    local typeLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    typeLabel:SetPoint("LEFT", name, "RIGHT", 6, 0)
    typeLabel:SetText(sourceData.sourceType)
    typeLabel:SetTextColor(color[1], color[2], color[3])

    -- Item count
    local itemCount = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    itemCount:SetPoint("BOTTOMLEFT", tag, "BOTTOMRIGHT", 8, 8)
    itemCount:SetText(string.format("%d item(s) missing", #sourceData.items))
    itemCount:SetTextColor(0.7, 0.7, 0.7)

    -- Score
    local score = row:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
    score:SetPoint("RIGHT", row, "RIGHT", -10, 4)
    score:SetText(string.format("%.1f", sourceData.score))
    score:SetTextColor(1.0, 0.82, 0.0)

    local scoreLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    scoreLabel:SetPoint("RIGHT", row, "RIGHT", -10, -10)
    scoreLabel:SetText("score")
    scoreLabel:SetTextColor(0.5, 0.5, 0.5)

    -- Chevron
    local chevron = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    chevron:SetPoint("BOTTOMRIGHT", row, "BOTTOMRIGHT", -10, 8)
    chevron:SetText("▼")
    chevron:SetTextColor(0.5, 0.5, 0.5)

    -- Score bar
    local barBg = row:CreateTexture(nil, "BACKGROUND")
    barBg:SetPoint("BOTTOMLEFT", row, "BOTTOMLEFT", 36, 5)
    barBg:SetPoint("BOTTOMRIGHT", row, "BOTTOMRIGHT", -60, 5)
    barBg:SetHeight(3)
    barBg:SetColorTexture(0.2, 0.2, 0.2, 0.8)

    local maxScore = GearPath.PriorityEngine.rankedSources[1]
        and GearPath.PriorityEngine.rankedSources[1].score or 1
    local barFill = row:CreateTexture(nil, "ARTWORK")
    barFill:SetPoint("LEFT", barBg, "LEFT", 0, 0)
    barFill:SetHeight(3)
    barFill:SetWidth(math.max(1, 200 * (sourceData.score / maxScore)))
    barFill:SetColorTexture(color[1], color[2], color[3], 0.8)

    -- Item rows creation (lazy, on first expand)
    local function createItemRows()
        for j, item in ipairs(sourceData.items) do
            local itemRow = CreateFrame("Frame", nil, parent, "BackdropTemplate")
            itemRow:SetPoint("TOPLEFT", row, "BOTTOMLEFT", 0, -((j - 1) * ITEM_HEIGHT))
            itemRow:SetPoint("TOPRIGHT", row, "BOTTOMRIGHT", 0, -((j - 1) * ITEM_HEIGHT))
            itemRow:SetHeight(ITEM_HEIGHT)
            itemRow:SetBackdrop({
                bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
                tile   = true, tileSize = 16,
                insets = { left = 3, right = 3, top = 3, bottom = 3 },
            })
            itemRow:SetBackdropColor(0.05, 0.05, 0.08, 0.95)

            -- Status dot
            local dot = itemRow:CreateTexture(nil, "ARTWORK")
            dot:SetSize(6, 6)
            dot:SetPoint("LEFT", itemRow, "LEFT", 36, 0)
            if item.status == "MISSING" then
                dot:SetColorTexture(1.0, 0.3, 0.3, 1)
            else
                dot:SetColorTexture(1.0, 0.82, 0.0, 1)
            end

            -- Item name
            local itemName = itemRow:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            itemName:SetPoint("LEFT", dot, "RIGHT", 6, 0)
            itemName:SetText(item.bisItem.itemName)
            itemName:SetTextColor(0.9, 0.9, 0.9)

            -- Boss name
            if item.bisItem.bossName then
                local bossLabel = itemRow:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
                bossLabel:SetPoint("RIGHT", itemRow, "RIGHT", -6, 0)
                bossLabel:SetText(item.bisItem.bossName)
                bossLabel:SetTextColor(0.5, 0.5, 0.5)
            end

            itemRow:Hide()
            table.insert(expandedItems, itemRow)
        end
    end

    -- Toggle expand on click
    row:SetScript("OnClick", function()
        if not expanded then
            if #expandedItems == 0 then
                createItemRows()
            end
            for _, itemRow in ipairs(expandedItems) do
                itemRow:Show()
            end
            row:SetHeight(ROW_HEIGHT + #expandedItems * ITEM_HEIGHT)
            chevron:SetText("▲")
            expanded = true
        else
            for _, itemRow in ipairs(expandedItems) do
                itemRow:Hide()
            end
            row:SetHeight(ROW_HEIGHT)
            chevron:SetText("▼")
            expanded = false
        end
        PriorityTab:RepositionRows()
    end)

    row:SetScript("OnEnter", function(r)
        r:SetBackdropColor(0.12, 0.12, 0.18, 0.95)
    end)
    row:SetScript("OnLeave", function(r)
        r:SetBackdropColor(0.08, 0.08, 0.12, 0.9)
    end)

    return row
end

function PriorityTab:ShowEmptyState()
    if container.emptyLabel then
        container.emptyLabel:Show()
        return
    end
    local label = container:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    label:SetPoint("CENTER", container, "CENTER", 0, 0)
    label:SetText("No missing BiS items!\nYou're fully geared for this spec.")
    label:SetTextColor(0.3, 1.0, 0.3)
    label:SetJustifyH("CENTER")
    container.emptyLabel = label
end