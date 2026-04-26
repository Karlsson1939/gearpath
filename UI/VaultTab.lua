-- GearPath
-- UI/VaultTab.lua - Great Vault advisor view

GearPath.VaultTab = {}
local VaultTab = GearPath.VaultTab

local container = nil

local VAULT_COLOR_KEYS = {
    [1] = "sourceDungeon",  -- M+
    [2] = "sourceRaid",     -- Raid
    [3] = "sourceWorld",    -- World / Delve
}

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

    local T = GearPath.Theme

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

    local yOffset = -T.space.sm
    yOffset = self:DrawProgressSection(child, advisor, yOffset)
    yOffset = yOffset - T.space.md
    yOffset = self:DrawRecommendationsSection(child, advisor, yOffset)

    child:SetHeight(math.abs(yOffset) + T.space.lg)
end

-- ============================================================
-- Progress section — shows all 9 vault slots with progress bars
-- ============================================================

function VaultTab:DrawProgressSection(child, advisor, yOffset)
    local T = GearPath.Theme

    -- Section header
    local header = child:CreateFontString(nil, "OVERLAY", T.font.label)
    header:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    header:SetText("Weekly Progress")
    header:SetTextColor(unpack(T.color.accentGold))
    yOffset = yOffset - 18  -- post-header gap

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
    local T = GearPath.Theme
    local color = T.color[VAULT_COLOR_KEYS[typeID]] or T.color.textPrimary
    local label = slots[1] and slots[1].typeLabel or "Unknown"

    -- Category label
    local catLabel = child:CreateFontString(nil, "OVERLAY", T.font.body)
    catLabel:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    catLabel:SetText(label)
    catLabel:SetTextColor(color[1], color[2], color[3])
    yOffset = yOffset - 18  -- post-header gap

    -- Sort slots by index
    table.sort(slots, function(a, b) return a.slotIndex < b.slotIndex end)

    for _, slot in ipairs(slots) do
        -- Slot background
        local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
        row:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
        row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -T.space.xs, yOffset)
        row:SetHeight(T.size.rowHeightVault)
        row:SetBackdrop({
            bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
            tile = true, tileSize = 16,
            insets = { left = 2, right = 2, top = 2, bottom = 2 },
        })
        row:SetBackdropColor(unpack(T.color.bgSurface))

        -- Slot number
        local slotNum = row:CreateFontString(nil, "OVERLAY", T.font.body)
        slotNum:SetPoint("LEFT", row, "LEFT", 6, 0)
        slotNum:SetText("Slot " .. slot.slotIndex)
        slotNum:SetTextColor(unpack(T.color.textMuted))
        slotNum:SetWidth(44)

        -- Lock/unlock icon
        local lockIcon = row:CreateFontString(nil, "OVERLAY", T.font.body)
        lockIcon:SetPoint("LEFT", slotNum, "RIGHT", T.space.xs, 0)
        if slot.unlocked then
            lockIcon:SetText("✓")
            lockIcon:SetTextColor(unpack(T.color.accentGold))
        else
            lockIcon:SetText("✗")
            lockIcon:SetTextColor(unpack(T.color.textMuted))
        end
        lockIcon:SetWidth(T.size.statusIcon)

        -- Progress bar background
        local barBg = row:CreateTexture(nil, "BACKGROUND")
        barBg:SetPoint("LEFT", lockIcon, "RIGHT", 6, 0)
        barBg:SetPoint("RIGHT", row, "RIGHT", -70, 0)
        barBg:SetHeight(T.size.barHeightSmall)
        barBg:SetColorTexture(0.15, 0.15, 0.15, 1)

        -- Progress bar fill
        local pct = slot.threshold > 0 and math.min(slot.progress / slot.threshold, 1) or 0
        local barWidth = math.max(barBg:GetWidth() or 100, 10)

        local barFill = row:CreateTexture(nil, "ARTWORK")
        barFill:SetPoint("LEFT", barBg, "LEFT", 0, 0)
        barFill:SetHeight(T.size.barHeightSmall)
        if slot.unlocked then
            barFill:SetColorTexture(color[1] * 0.8, color[2] * 0.8, color[3] * 0.8, 0.9)
        else
            barFill:SetColorTexture(color[1] * 0.5, color[2] * 0.5, color[3] * 0.5, 0.7)
        end
        -- Width set via OnSizeChanged to get real pixel width
        row:SetScript("OnSizeChanged", function()
            local w = barBg:GetWidth()
            if w and w > 0 then
                barFill:SetWidth(math.max(1, w * pct))
            end
        end)

        -- Progress text
        local progText = row:CreateFontString(nil, "OVERLAY", T.font.body)
        progText:SetPoint("LEFT", barBg, "LEFT", T.space.xs, 0)
        progText:SetText(string.format("%d / %d", slot.progress, slot.threshold))
        progText:SetTextColor(unpack(T.color.textSecondary))

        -- Ilvl label
        local ilvlText = row:CreateFontString(nil, "OVERLAY", T.font.body)
        ilvlText:SetPoint("RIGHT", row, "RIGHT", -6, 0)
        if slot.unlocked then
            ilvlText:SetText(string.format("ilvl %d", slot.itemLevel))
            ilvlText:SetTextColor(unpack(T.color.accentGold))
        else
            ilvlText:SetText(string.format("ilvl %d", slot.itemLevel))
            ilvlText:SetTextColor(unpack(T.color.textDisabled))
        end

        yOffset = yOffset - (T.size.rowHeightVault + 2)
    end

    return yOffset
end

-- ============================================================
-- Recommendations section
-- ============================================================

function VaultTab:DrawRecommendationsSection(child, advisor, yOffset)
    local T = GearPath.Theme

    local header = child:CreateFontString(nil, "OVERLAY", T.font.label)
    header:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    header:SetText("Vault Recommendations")
    header:SetTextColor(unpack(T.color.accentGold))
    yOffset = yOffset - 18  -- post-header gap

    local recs = advisor:GetRankedRecommendations()

    local hasAny = false
    for _, rec in ipairs(recs) do
        if rec.slot.unlocked then
            hasAny = true
            yOffset = self:DrawRecommendationRow(child, rec, yOffset)
            yOffset = yOffset - T.space.xs
        end
    end

    -- Locked slots needing progress
    local needsHeader = false
    for _, rec in ipairs(recs) do
        if not rec.slot.unlocked then
            if not needsHeader then
                yOffset = yOffset - T.space.sm
                local subHeader = child:CreateFontString(nil, "OVERLAY", T.font.body)
                subHeader:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
                subHeader:SetText("Still Locked — Complete More Activities")
                subHeader:SetTextColor(unpack(T.color.textMuted))
                yOffset = yOffset - 18  -- post-header gap
                needsHeader = true
            end
            yOffset = self:DrawLockedRow(child, rec, yOffset)
            yOffset = yOffset - T.space.xs
        end
    end

    if not hasAny and not needsHeader then
        local empty = child:CreateFontString(nil, "OVERLAY", T.font.label)
        empty:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
        empty:SetText("No vault slots unlocked this week yet.")
        empty:SetTextColor(unpack(T.color.textMuted))
        yOffset = yOffset - T.size.rowHeightStandard
    end

    return yOffset
end

function VaultTab:DrawRecommendationRow(child, rec, yOffset)
    local T = GearPath.Theme
    local slot  = rec.slot
    local color = T.color[VAULT_COLOR_KEYS[slot.type]] or T.color.textPrimary

    local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
    row:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -T.space.xs, yOffset)

    local matchCount = #rec.bisMatches
    local rowHeight  = 28 + (math.min(matchCount, 3) * 18)
    row:SetHeight(rowHeight)
    row:SetBackdrop({
        bgFile   = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        edgeFile = T.border.row.edgeFile,
        tile = true, tileSize = 16, edgeSize = 10,
        insets = { left = T.border.row.insets[1], right = T.border.row.insets[2],
                   top = T.border.row.insets[3], bottom = T.border.row.insets[4] },
    })
    row:SetBackdropColor(unpack(T.color.bgElevated))

    -- Color strip
    local strip = row:CreateTexture(nil, "ARTWORK")
    strip:SetPoint("TOPLEFT", row, "TOPLEFT", 3, -3)
    strip:SetPoint("BOTTOMLEFT", row, "BOTTOMLEFT", 3, 3)
    strip:SetWidth(3)
    strip:SetColorTexture(color[1], color[2], color[3], 0.9)

    -- Title: "M+ Slot 1 — ilvl 252"
    local title = row:CreateFontString(nil, "OVERLAY", T.font.label)
    title:SetPoint("TOPLEFT", strip, "TOPRIGHT", T.space.sm, -6)
    title:SetText(string.format("%s  Slot %d", slot.typeLabel, slot.slotIndex))
    title:SetTextColor(unpack(T.color.textPrimary))

    local ilvlLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    ilvlLabel:SetPoint("LEFT", title, "RIGHT", T.space.sm, 0)
    ilvlLabel:SetText(string.format("ilvl %d", slot.itemLevel))
    ilvlLabel:SetTextColor(unpack(T.color.accentGold))

    -- BiS matches
    if matchCount == 0 then
        local noMatch = row:CreateFontString(nil, "OVERLAY", T.font.body)
        noMatch:SetPoint("TOPLEFT", title, "BOTTOMLEFT", 0, -T.space.xs)
        noMatch:SetText("No missing BiS items for this vault type")
        noMatch:SetTextColor(unpack(T.color.textMuted))
    else
        local countLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
        countLabel:SetPoint("RIGHT", row, "RIGHT", -T.space.sm, 0)
        countLabel:SetText(matchCount .. " BiS match" .. (matchCount ~= 1 and "es" or ""))
        countLabel:SetTextColor(color[1], color[2], color[3])

        local prevAnchor = title
        for i, match in ipairs(rec.bisMatches) do
            if i > 3 then break end
            local itemLine = row:CreateFontString(nil, "OVERLAY", T.font.body)
            itemLine:SetPoint("TOPLEFT", prevAnchor, "BOTTOMLEFT", 0, -2)
            local statusColor = match.status == "MISSING" and T.color.statusMissing or T.color.statusUpgradeable
            itemLine:SetText(string.format("  • %s", match.bisItem.itemName))
            itemLine:SetTextColor(unpack(statusColor))
            prevAnchor = itemLine
        end
        if matchCount > 3 then
            local more = row:CreateFontString(nil, "OVERLAY", T.font.body)
            more:SetPoint("TOPLEFT", prevAnchor, "BOTTOMLEFT", 0, -2)
            more:SetText(string.format("  + %d more...", matchCount - 3))
            more:SetTextColor(unpack(T.color.textMuted))
        end
    end

    yOffset = yOffset - (rowHeight + 2)
    return yOffset
end

function VaultTab:DrawLockedRow(child, rec, yOffset)
    local T = GearPath.Theme
    local slot  = rec.slot
    local color = T.color[VAULT_COLOR_KEYS[slot.type]] or T.color.textPrimary

    local row = CreateFrame("Frame", nil, child, "BackdropTemplate")
    row:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", -T.space.xs, yOffset)
    row:SetHeight(T.size.rowHeightVault)
    row:SetBackdrop({
        bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        tile = true, tileSize = 16,
        insets = { left = T.border.rowFlat.insets[1], right = T.border.rowFlat.insets[2],
                   top = T.border.rowFlat.insets[3], bottom = T.border.rowFlat.insets[4] },
    })
    row:SetBackdropColor(unpack(T.color.bgDisabled))

    local label = row:CreateFontString(nil, "OVERLAY", T.font.body)
    label:SetPoint("LEFT", row, "LEFT", T.space.sm, 0)
    label:SetText(string.format("%s Slot %d — %d / %d needed",
        slot.typeLabel, slot.slotIndex, slot.progress, slot.threshold))
    label:SetTextColor(unpack(T.color.textDisabled))

    local ilvlLabel = row:CreateFontString(nil, "OVERLAY", T.font.body)
    ilvlLabel:SetPoint("RIGHT", row, "RIGHT", -T.space.sm, 0)
    ilvlLabel:SetText(string.format("ilvl %d", slot.itemLevel))
    ilvlLabel:SetTextColor(unpack(T.color.textDisabled))

    yOffset = yOffset - (T.size.rowHeightVault + 2)
    return yOffset
end

function VaultTab:ShowUnavailable(child)
    local T = GearPath.Theme
    local label = child:CreateFontString(nil, "OVERLAY", T.font.label)
    label:SetPoint("TOP", child, "TOP", 0, -40)
    label:SetText("Vault data unavailable.\n\nOpen the Great Vault (H) to load your weekly progress,\nthen reopen GearPath.")
    label:SetTextColor(unpack(T.color.textMuted))
    label:SetJustifyH("CENTER")
end
