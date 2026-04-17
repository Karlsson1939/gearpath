-- GearPath
-- UI/VaultTab.lua - Great Vault advisor view

GearPath.VaultTab = {}
local VaultTab = GearPath.VaultTab

local container = nil

local TYPE_COLORS = {
    [1] = { 0.4, 0.8, 1.0 },  -- M+    blue
    [2] = { 1.0, 0.5, 0.2 },  -- Raid  orange
    [3] = { 0.6, 1.0, 0.4 },  -- World green
}

local LOCK_COLOR   = { 0.5, 0.5, 0.5 }
local UNLOCK_COLOR = { 1.0, 0.82, 0.0 }

function VaultTab:Show(parent)
    if not container then
        container = CreateFrame("ScrollFrame", "GearPathVaultTab", parent, "UIPanelScrollFrameTemplate")
        container:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, 0)
        container:SetPoint("BOTTOMRIGHT", parent, "BOTTOMRIGHT", -20, 0)

        container.child = CreateFrame("Frame", nil, container)
        container.child:SetSize(parent:GetWidth() - 20, 800)
        container:SetScrollChild(container.child)
    else
        container:SetParent(parent)
        container:SetPoint("TOPLEFT", parent, "TOPLEFT", 0, 0)
        container:SetPoint("BOTTOMRIGHT", parent, "BOTTOMRIGHT", -20, 0)
    end

    container:Show()
    self:Refresh()
end

function VaultTab:Hide()
    if container then container:Hide() end
end

function VaultTab:Refresh()
    if not container or not container:IsShown() then return end

    local child = container.child
    -- Clear all children
    for _, f in pairs({ child:GetChildren() }) do
        f:Hide()
    end
    for _, r in pairs({ child:GetRegions() }) do
        r:Hide()
    end

    local advisor = GearPath.VaultAdvisor
    if not advisor then return end

    advisor:Refresh()

    if not advisor.available then
        self:ShowUnavailable(child)
        return
    end

    local yOffset = -8
    yOffset = self:DrawProgressSection(child, advisor, yOffset)
    yOffset = yOffset - 12
    yOffset = self:DrawRecommendationsSection(child, advisor, yOffset)

    child:SetHeight(math.abs(yOffset) + 16)
end

-- ============================================================
-- Progress section — shows all 9 vault slots with progress bars
-- ============================================================

function VaultTab:DrawProgressSection(child, advisor, yOffset)
    -- Section header
    local header = child:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    header:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    header:SetText("WEEKLY PROGRESS")
    header:SetTextColor(0.6, 0.6, 0.6)
    yOffset = yOffset - 18

    -- Group slots by type
    local byType = {}
    for _, slot in ipairs(advisor.slots) do
        if not byType[slot.type] then byType[slot.type] = {} end
        table.insert(byType[slot.type], slot)
    end

    local typeOrder = { 1, 2, 3 }
    for _, typeID in ipairs(typeOrder) do
        local slots = byType[typeID]
        if slots then
            yOffset = self:DrawActivityRow(child, typeID, slots, yOffset)
            yOffset = yOffset - 6
        end
    end

    return yOffset
end

function VaultTab:DrawActivityRow(child, typeID, slots, yOffset)
    local color = TYPE_COLORS[typeID] or { 1, 1, 1 }
    local label = slots[1] and slots[1].typeLabel or "Unknown"

    -- Category label
    local catLabel = child:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    catLabel:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    catLabel:SetText(label)
    catLabel:SetTextColor(color[1], color[2], color[3])
    yOffset = yOffset - 18

    -- Sort slots by index
    table.sort(slots, function(a, b) return a.slotIndex < b.slotIndex end)

    for _, slot in ipairs(slots) do
        -- Slot background
        local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
        row:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
        row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -4, yOffset)
        row:SetHeight(24)
        row:SetBackdrop({
            bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
            tile = true, tileSize = 16,
            insets = { left = 2, right = 2, top = 2, bottom = 2 },
        })
        row:SetBackdropColor(0.08, 0.08, 0.12, 0.9)

        -- Slot number
        local slotNum = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        slotNum:SetPoint("LEFT", row, "LEFT", 6, 0)
        slotNum:SetText("Slot " .. slot.slotIndex)
        slotNum:SetTextColor(0.6, 0.6, 0.6)
        slotNum:SetWidth(44)

        -- Lock/unlock icon
        local lockIcon = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        lockIcon:SetPoint("LEFT", slotNum, "RIGHT", 4, 0)
        if slot.unlocked then
            lockIcon:SetText("✓")
            lockIcon:SetTextColor(UNLOCK_COLOR[1], UNLOCK_COLOR[2], UNLOCK_COLOR[3])
        else
            lockIcon:SetText("✗")
            lockIcon:SetTextColor(LOCK_COLOR[1], LOCK_COLOR[2], LOCK_COLOR[3])
        end
        lockIcon:SetWidth(14)

        -- Progress bar background
        local barBg = row:CreateTexture(nil, "BACKGROUND")
        barBg:SetPoint("LEFT", lockIcon, "RIGHT", 6, 0)
        barBg:SetPoint("RIGHT", row, "RIGHT", -70, 0)
        barBg:SetHeight(8)
        barBg:SetColorTexture(0.15, 0.15, 0.15, 1)

        -- Progress bar fill
        local pct = slot.threshold > 0 and math.min(slot.progress / slot.threshold, 1) or 0
        local barWidth = math.max(barBg:GetWidth() or 100, 10)

        local barFill = row:CreateTexture(nil, "ARTWORK")
        barFill:SetPoint("LEFT", barBg, "LEFT", 0, 0)
        barFill:SetHeight(8)
        if slot.unlocked then
            barFill:SetColorTexture(color[1] * 0.8, color[2] * 0.8, color[3] * 0.8, 0.9)
        else
            barFill:SetColorTexture(color[1] * 0.5, color[2] * 0.5, color[3] * 0.5, 0.7)
        end
        -- Width set via OnSizeChanged to get real pixel width
        barBg:SetScript("OnSizeChanged", function(self, w, h)
            barFill:SetWidth(math.max(1, w * pct))
        end)

        -- Progress text
        local progText = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        progText:SetPoint("LEFT", barBg, "LEFT", 4, 0)
        progText:SetText(string.format("%d / %d", slot.progress, slot.threshold))
        progText:SetTextColor(0.9, 0.9, 0.9)

        -- Ilvl label
        local ilvlText = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        ilvlText:SetPoint("RIGHT", row, "RIGHT", -6, 0)
        if slot.unlocked then
            ilvlText:SetText(string.format("ilvl %d", slot.itemLevel))
            ilvlText:SetTextColor(UNLOCK_COLOR[1], UNLOCK_COLOR[2], UNLOCK_COLOR[3])
        else
            ilvlText:SetText(string.format("ilvl %d", slot.itemLevel))
            ilvlText:SetTextColor(0.4, 0.4, 0.4)
        end

        yOffset = yOffset - 26
    end

    return yOffset
end

-- ============================================================
-- Recommendations section
-- ============================================================

function VaultTab:DrawRecommendationsSection(child, advisor, yOffset)
    local header = child:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    header:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    header:SetText("VAULT RECOMMENDATIONS")
    header:SetTextColor(0.6, 0.6, 0.6)
    yOffset = yOffset - 18

    local recs = advisor:GetRankedRecommendations()

    local hasAny = false
    for _, rec in ipairs(recs) do
        if rec.slot.unlocked then
            hasAny = true
            yOffset = self:DrawRecommendationRow(child, rec, yOffset)
            yOffset = yOffset - 4
        end
    end

    -- Locked slots needing progress
    local needsHeader = false
    for _, rec in ipairs(recs) do
        if not rec.slot.unlocked then
            if not needsHeader then
                yOffset = yOffset - 8
                local subHeader = child:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
                subHeader:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
                subHeader:SetText("STILL LOCKED — COMPLETE MORE ACTIVITIES")
                subHeader:SetTextColor(0.5, 0.5, 0.5)
                yOffset = yOffset - 18
                needsHeader = true
            end
            yOffset = self:DrawLockedRow(child, rec, yOffset)
            yOffset = yOffset - 4
        end
    end

    if not hasAny and not needsHeader then
        local empty = child:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        empty:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
        empty:SetText("No vault slots unlocked this week yet.")
        empty:SetTextColor(0.5, 0.5, 0.5)
        yOffset = yOffset - 22
    end

    return yOffset
end

function VaultTab:DrawRecommendationRow(child, rec, yOffset)
    local slot  = rec.slot
    local color = TYPE_COLORS[slot.type] or { 1, 1, 1 }

    local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
    row:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -4, yOffset)

    local matchCount = #rec.bisMatches
    local rowHeight  = 28 + (math.min(matchCount, 3) * 18)
    row:SetHeight(rowHeight)
    row:SetBackdrop({
        bgFile   = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border",
        tile = true, tileSize = 16, edgeSize = 10,
        insets = { left = 3, right = 3, top = 3, bottom = 3 },
    })
    row:SetBackdropColor(0.08, 0.10, 0.14, 0.95)

    -- Color strip
    local strip = row:CreateTexture(nil, "ARTWORK")
    strip:SetPoint("TOPLEFT", row, "TOPLEFT", 3, -3)
    strip:SetPoint("BOTTOMLEFT", row, "BOTTOMLEFT", 3, 3)
    strip:SetWidth(3)
    strip:SetColorTexture(color[1], color[2], color[3], 0.9)

    -- Title: "M+ Slot 1 — ilvl 252"
    local title = row:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    title:SetPoint("TOPLEFT", strip, "TOPRIGHT", 8, -6)
    title:SetText(string.format("%s  Slot %d", slot.typeLabel, slot.slotIndex))
    title:SetTextColor(1, 1, 1)

    local ilvlLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    ilvlLabel:SetPoint("LEFT", title, "RIGHT", 8, 0)
    ilvlLabel:SetText(string.format("ilvl %d", slot.itemLevel))
    ilvlLabel:SetTextColor(UNLOCK_COLOR[1], UNLOCK_COLOR[2], UNLOCK_COLOR[3])

    -- BiS matches
    if matchCount == 0 then
        local noMatch = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        noMatch:SetPoint("TOPLEFT", title, "BOTTOMLEFT", 0, -4)
        noMatch:SetText("No missing BiS items for this vault type")
        noMatch:SetTextColor(0.5, 0.5, 0.5)
    else
        local countLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        countLabel:SetPoint("RIGHT", row, "RIGHT", -8, 0)
        countLabel:SetText(matchCount .. " BiS match" .. (matchCount ~= 1 and "es" or ""))
        countLabel:SetTextColor(color[1], color[2], color[3])

        local prevAnchor = title
        for i, match in ipairs(rec.bisMatches) do
            if i > 3 then break end
            local itemLine = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            itemLine:SetPoint("TOPLEFT", prevAnchor, "BOTTOMLEFT", 0, -2)
            local statusColor = match.status == "MISSING" and { 1.0, 0.3, 0.3 } or { 1.0, 0.82, 0.0 }
            itemLine:SetText(string.format("  • %s", match.bisItem.itemName))
            itemLine:SetTextColor(statusColor[1], statusColor[2], statusColor[3])
            prevAnchor = itemLine
        end
        if matchCount > 3 then
            local more = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            more:SetPoint("TOPLEFT", prevAnchor, "BOTTOMLEFT", 0, -2)
            more:SetText(string.format("  + %d more...", matchCount - 3))
            more:SetTextColor(0.5, 0.5, 0.5)
        end
    end

    yOffset = yOffset - (rowHeight + 2)
    return yOffset
end

function VaultTab:DrawLockedRow(child, rec, yOffset)
    local slot  = rec.slot
    local color = TYPE_COLORS[slot.type] or { 1, 1, 1 }

    local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
    row:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -4, yOffset)
    row:SetHeight(24)
    row:SetBackdrop({
        bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        tile = true, tileSize = 16,
        insets = { left = 3, right = 3, top = 3, bottom = 3 },
    })
    row:SetBackdropColor(0.06, 0.06, 0.08, 0.8)

    local label = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    label:SetPoint("LEFT", row, "LEFT", 8, 0)
    label:SetText(string.format("%s Slot %d — %d / %d needed",
        slot.typeLabel, slot.slotIndex, slot.progress, slot.threshold))
    label:SetTextColor(0.45, 0.45, 0.45)

    local ilvlLabel = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    ilvlLabel:SetPoint("RIGHT", row, "RIGHT", -8, 0)
    ilvlLabel:SetText(string.format("ilvl %d", slot.itemLevel))
    ilvlLabel:SetTextColor(0.35, 0.35, 0.35)

    yOffset = yOffset - 26
    return yOffset
end

function VaultTab:ShowUnavailable(child)
    local label = child:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    label:SetPoint("TOP", child, "TOP", 0, -40)
    label:SetText("Vault data unavailable.\n\nOpen the Great Vault (H) to load your weekly progress,\nthen reopen GearPath.")
    label:SetTextColor(0.5, 0.5, 0.5)
    label:SetJustifyH("CENTER")
end