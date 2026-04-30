-- GearPath
-- UI/VaultTab.lua - Top upgrade gaps shortlist for Vault picks

GearPath.VaultTab = {}
local VaultTab = GearPath.VaultTab

local container = nil
local rows = {}

function VaultTab:Show(parent)
    if not container then
        container = CreateFrame("Frame", "GearPathVaultTab", parent)
        container:SetAllPoints(parent)

        container.scrollFrame = CreateFrame("ScrollFrame", nil, container, "UIPanelScrollFrameTemplate")
        container.scrollFrame:SetPoint("TOPLEFT", container, "TOPLEFT", 0, -22)
        container.scrollFrame:SetPoint("BOTTOMRIGHT", container, "BOTTOMRIGHT", -20, 0)

        container.scrollChild = CreateFrame("Frame", nil, container.scrollFrame)
        container.scrollChild:SetSize(parent:GetWidth() - 20, 1)
        container.scrollFrame:SetScrollChild(container.scrollChild)
    else
        container:SetParent(parent)
        container:SetAllPoints(parent)
    end
    container:Show()
    self:Refresh()
end

function VaultTab:Hide()
    if container then container:Hide() end
end

function VaultTab:Refresh()
    if not container or not container:IsShown() then return end

    local T = GearPath.Theme

    -- Clear existing rows
    for _, row in ipairs(rows) do
        row:Hide()
    end
    rows = {}

    -- Hide header/empty elements from previous render
    if container.header then container.header:Hide() end
    if container.emptyLabel then container.emptyLabel:Hide() end

    -- Not-ready guard
    if not GearPath.Detection or not GearPath.Detection:IsReady() then
        self:ShowNotReady()
        return
    end

    local shortlist = GearPath.UpgradeShortlist
    if not shortlist then return end

    local topItems = shortlist.topItems

    -- Empty state
    if #topItems == 0 then
        self:ShowEmptyState()
        return
    end

    -- Header
    if not container.header then
        container.header = container:CreateFontString(nil, "OVERLAY", T.font.body)
        container.header:SetPoint("TOPLEFT", container, "TOPLEFT", T.space.xs, -T.space.xs)
    end
    container.header:Show()
    local specSummary = GearPath.Detection:GetSummary()
    container.header:SetText("Top upgrades for " .. specSummary)
    container.header:SetTextColor(unpack(T.color.textMuted))

    -- Render rows
    local yOffset = 0
    for i, entry in ipairs(topItems) do
        local row = self:CreateItemRow(container.scrollChild, entry, i, yOffset)
        table.insert(rows, row)
        yOffset = yOffset - (row:GetHeight() + 4)
    end
    container.scrollChild:SetHeight(math.abs(yOffset))
end

-- ============================================================
-- Row rendering
-- ============================================================

local SOURCE_COLOR_KEYS = {
    DUNGEON = "sourceDungeon",
    RAID    = "sourceRaid",
    CRAFTED = "sourceCrafted",
    WORLD   = "sourceWorld",
    DELVE   = "sourceDelve",
    PVP     = "sourcePvP",
}

function VaultTab:CreateItemRow(parent, entry, index, yOffset)
    local T = GearPath.Theme
    local ROW_HEIGHT = 42

    local row = CreateFrame("Frame", nil, parent, "BackdropTemplate")
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

    -- Status-keyed color strip on left edge
    local statusColor
    if entry.status == "MISSING" then
        statusColor = T.color.statusMissing
    else
        statusColor = T.color.statusUpgradeable
    end

    local strip = row:CreateTexture(nil, "ARTWORK")
    strip:SetPoint("TOPLEFT", row, "TOPLEFT", 3, -3)
    strip:SetPoint("BOTTOMLEFT", row, "BOTTOMLEFT", 3, 3)
    strip:SetWidth(3)
    strip:SetColorTexture(statusColor[1], statusColor[2], statusColor[3], 0.9)

    -- Rank number
    local rank = row:CreateFontString(nil, "OVERLAY", T.font.body)
    rank:SetPoint("TOPLEFT", strip, "TOPRIGHT", T.space.xs, -4)
    rank:SetText(index .. ".")
    rank:SetTextColor(unpack(T.color.textMuted))
    rank:SetWidth(18)

    -- Slot name (small, dim)
    local slotLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    slotLabel:SetPoint("LEFT", rank, "RIGHT", 2, 0)
    slotLabel:SetText(entry.slotName)
    slotLabel:SetTextColor(unpack(T.color.textMuted))

    -- Item name (primary text)
    local itemName = row:CreateFontString(nil, "OVERLAY", T.font.label)
    itemName:SetPoint("TOPLEFT", slotLabel, "TOPRIGHT", T.space.sm, 0)
    itemName:SetPoint("RIGHT", row, "RIGHT", -60, 0)
    itemName:SetText(entry.bisItem.itemName)
    itemName:SetTextColor(unpack(T.color.textPrimary))
    itemName:SetJustifyH("LEFT")
    itemName:SetWordWrap(false)

    -- Source name (second line, small, dim)
    local sourceColor = T.color[SOURCE_COLOR_KEYS[entry.bisItem.sourceType]] or T.color.textMuted
    local sourceLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    sourceLabel:SetPoint("BOTTOMLEFT", rank, "BOTTOMLEFT", 0, 4)
    sourceLabel:SetText(entry.bisItem.sourceName)
    sourceLabel:SetTextColor(sourceColor[1], sourceColor[2], sourceColor[3])

    -- Score (right-aligned, gold)
    local score = row:CreateFontString(nil, "OVERLAY", T.font.emphasis)
    score:SetPoint("TOPRIGHT", row, "TOPRIGHT", -10, -8)
    score:SetText(string.format("%.1f", entry.score))
    score:SetTextColor(unpack(T.color.accentGold))

    local scoreLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    scoreLabel:SetPoint("TOPRIGHT", row, "TOPRIGHT", -10, -22)
    scoreLabel:SetText("score")
    scoreLabel:SetTextColor(unpack(T.color.textMuted))

    -- Status dot
    local dot = row:CreateTexture(nil, "ARTWORK")
    dot:SetSize(6, 6)
    dot:SetPoint("RIGHT", score, "LEFT", -6, 0)
    dot:SetColorTexture(statusColor[1], statusColor[2], statusColor[3], 1)

    return row
end

-- ============================================================
-- State screens
-- ============================================================

function VaultTab:ShowNotReady()
    local T = GearPath.Theme
    if not container.emptyLabel then
        container.emptyLabel = container:CreateFontString(nil, "OVERLAY", T.font.label)
        container.emptyLabel:SetPoint("CENTER", container, "CENTER", 0, 0)
        container.emptyLabel:SetJustifyH("CENTER")
    end
    local msg = GearPath.Detection:GetNotReadyMessage() or "GearPath is still loading."
    container.emptyLabel:SetText(msg)
    container.emptyLabel:SetTextColor(unpack(T.color.textMuted))
    container.emptyLabel:Show()
end

function VaultTab:ShowEmptyState()
    local T = GearPath.Theme
    if not container.emptyLabel then
        container.emptyLabel = container:CreateFontString(nil, "OVERLAY", T.font.label)
        container.emptyLabel:SetPoint("CENTER", container, "CENTER", 0, 0)
        container.emptyLabel:SetJustifyH("CENTER")
    end
    container.emptyLabel:SetText("You're fully BiS for this spec.")
    container.emptyLabel:SetTextColor(unpack(T.color.statusEquipped))
    container.emptyLabel:Show()
end
