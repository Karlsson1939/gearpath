-- GearPath
-- UI/PriorityTab.lua - Dungeon priority list view

GearPath.PriorityTab = {}
local PriorityTab = GearPath.PriorityTab

local container = nil
local rows = {}

local SOURCE_COLOR_KEYS = {
    DUNGEON = "sourceDungeon",
    RAID    = "sourceRaid",
    CRAFTED = "sourceCrafted",
    WORLD   = "sourceWorld",
    DELVE   = "sourceDelve",
    PVP     = "sourcePvP",
}

local SOURCE_TYPE_DISPLAY = {
    DUNGEON = "Dungeon",
    RAID    = "Raid",
    CRAFTED = "Crafted",
    WORLD   = "World",
    DELVE   = "Delve",
    PVP     = "PvP",
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
    local T = GearPath.Theme

    if container.progressBg then
        container.progressBg:Show()
        container.progressFill:Show()
        container.progressLabel:Show()
        container.progressCount:Show()
        container.scoreExplanation:Show()
    else
        local bg = container:CreateTexture(nil, "BACKGROUND")
        bg:SetPoint("TOPLEFT", container, "TOPLEFT", 0, -T.space.xs)
        bg:SetPoint("TOPRIGHT", container, "TOPRIGHT", 0, -T.space.xs)
        bg:SetHeight(T.size.barHeight)
        bg:SetColorTexture(unpack(T.color.bgTrack))
        container.progressBg = bg

        local fill = container:CreateTexture(nil, "ARTWORK")
        fill:SetPoint("TOPLEFT", bg, "TOPLEFT", 1, -1)
        fill:SetHeight(T.size.barHeight - 2)
        local cc = T.color.classAccent
        fill:SetColorTexture(cc[1], cc[2], cc[3], 0.7)
        container.progressFill = fill

        local label = container:CreateFontString(nil, "OVERLAY", T.font.body)
        label:SetPoint("LEFT", bg, "LEFT", T.space.xs, 0)
        label:SetText("BiS completion")
        label:SetTextColor(unpack(T.color.textSecondary))
        container.progressLabel = label

        local count = container:CreateFontString(nil, "OVERLAY", T.font.body)
        count:SetPoint("RIGHT", bg, "RIGHT", -T.space.xs, 0)
        container.progressCount = count

        local explanation = container:CreateFontString(nil, "OVERLAY", T.font.body)
        explanation:SetPoint("TOPLEFT", bg, "BOTTOMLEFT", T.space.xs, -2)
        explanation:SetText("Sources ranked by missing BiS item priorities.")
        explanation:SetTextColor(unpack(T.color.textMuted))
        container.scoreExplanation = explanation
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
    container.progressCount:SetTextColor(unpack(T.color.accentGold))
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
    local T = GearPath.Theme
    local ROW_HEIGHT  = T.size.rowHeightLarge
    local ITEM_HEIGHT = T.size.rowHeightSmall
    local expandedItems = {}
    local expanded    = false

    local row = CreateFrame("Button", nil, parent, "BackdropTemplate")
    row:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, yOffset)
    row:SetPoint("TOPRIGHT", parent, "TOPRIGHT", 0, yOffset)
    row:SetHeight(ROW_HEIGHT)
    row:SetBackdrop({
        bgFile   = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        edgeFile = T.border.row.edgeFile,
        tile     = true, tileSize = 16, edgeSize = T.border.row.edgeSize,
        insets   = { left = T.border.row.insets[1], right = T.border.row.insets[2],
                     top = T.border.row.insets[3], bottom = T.border.row.insets[4] },
    })
    row:SetBackdropColor(unpack(T.color.bgSurface))

    local color = T.color[SOURCE_COLOR_KEYS[sourceData.sourceType]] or T.color.textPrimary

    -- Rank number
    local rank = row:CreateFontString(nil, "OVERLAY", T.font.emphasis)
    rank:SetPoint("LEFT", row, "LEFT", T.space.sm, T.space.xs)
    rank:SetText(index)
    rank:SetTextColor(unpack(T.color.textMuted))
    rank:SetWidth(20)

    -- Color tag
    local tag = row:CreateTexture(nil, "ARTWORK")
    tag:SetPoint("LEFT", rank, "RIGHT", 6, 0)
    tag:SetSize(3, 32)
    tag:SetColorTexture(color[1], color[2], color[3], 0.9)

    -- Source name
    local name = row:CreateFontString(nil, "OVERLAY", T.font.label)
    name:SetPoint("TOPLEFT", tag, "TOPRIGHT", T.space.sm, -T.space.xs)
    name:SetText(sourceData.sourceName)
    name:SetTextColor(unpack(T.color.textPrimary))

    -- Type badge
    local typeLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    typeLabel:SetPoint("LEFT", name, "RIGHT", 6, 0)
    typeLabel:SetText(SOURCE_TYPE_DISPLAY[sourceData.sourceType] or sourceData.sourceType)
    typeLabel:SetTextColor(color[1], color[2], color[3])

    -- Item count
    local itemCount = row:CreateFontString(nil, "OVERLAY", T.font.body)
    itemCount:SetPoint("BOTTOMLEFT", tag, "BOTTOMRIGHT", T.space.sm, T.space.sm)
    itemCount:SetText(string.format("%d item(s) missing", #sourceData.items))
    itemCount:SetTextColor(unpack(T.color.textMuted))

    -- Score
    local score = row:CreateFontString(nil, "OVERLAY", T.font.emphasis)
    score:SetPoint("RIGHT", row, "RIGHT", -10, 4)
    score:SetText(string.format("%.1f", sourceData.score))
    score:SetTextColor(unpack(T.color.accentGold))

    local scoreLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    scoreLabel:SetPoint("RIGHT", row, "RIGHT", -10, -10)
    scoreLabel:SetText("score")
    scoreLabel:SetTextColor(unpack(T.color.textMuted))

    -- Chevron
    local chevron = row:CreateFontString(nil, "OVERLAY", T.font.body)
    chevron:SetPoint("BOTTOMRIGHT", row, "BOTTOMRIGHT", -10, T.space.sm)
    chevron:SetText("▼")
    chevron:SetTextColor(unpack(T.color.textMuted))

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
                insets = { left = T.border.rowFlat.insets[1], right = T.border.rowFlat.insets[2],
                           top = T.border.rowFlat.insets[3], bottom = T.border.rowFlat.insets[4] },
            })
            itemRow:SetBackdropColor(unpack(T.color.bgBase))

            -- Status dot
            local dot = itemRow:CreateTexture(nil, "ARTWORK")
            dot:SetSize(6, 6)
            dot:SetPoint("LEFT", itemRow, "LEFT", 36, 0)
            if item.status == "MISSING" then
                local sm = T.color.statusMissing
                dot:SetColorTexture(sm[1], sm[2], sm[3], 1)
            else
                local su = T.color.statusUpgradeable
                dot:SetColorTexture(su[1], su[2], su[3], 1)
            end

            -- Item name
            local itemName = itemRow:CreateFontString(nil, "OVERLAY", T.font.body)
            itemName:SetPoint("LEFT", dot, "RIGHT", 6, 0)
            itemName:SetText(item.bisItem.itemName)
            itemName:SetTextColor(unpack(T.color.textSecondary))

            -- Boss name
            if item.bisItem.bossName then
                local bossLabel = itemRow:CreateFontString(nil, "OVERLAY", T.font.body)
                bossLabel:SetPoint("RIGHT", itemRow, "RIGHT", -6, 0)
                bossLabel:SetText(item.bisItem.bossName)
                bossLabel:SetTextColor(unpack(T.color.textMuted))
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
        r:SetBackdropColor(unpack(T.color.bgElevated))
    end)
    row:SetScript("OnLeave", function(r)
        r:SetBackdropColor(unpack(T.color.bgSurface))
    end)

    return row
end

function PriorityTab:ShowEmptyState()
    local T = GearPath.Theme
    if container.emptyLabel then
        container.emptyLabel:Show()
        return
    end
    local label = container:CreateFontString(nil, "OVERLAY", T.font.label)
    label:SetPoint("CENTER", container, "CENTER", 0, 0)
    label:SetText("No missing BiS items!\nYou're fully geared for this spec.")
    label:SetTextColor(unpack(T.color.statusEquipped))
    label:SetJustifyH("CENTER")
    container.emptyLabel = label
end
