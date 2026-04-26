-- GearPath
-- UI/StatsTab.lua - Stats & Consumables reference view

GearPath.StatsTab = {}
local StatsTab = GearPath.StatsTab

local container = nil

-- ============================================================
-- Helpers
-- ============================================================

local function AddHeader(child, yOffset, text)
    local T = GearPath.Theme
    local header = child:CreateFontString(nil, "OVERLAY", T.font.label)
    header:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, yOffset)
    header:SetText(text)
    header:SetTextColor(unpack(T.color.accentGold))
    return yOffset - T.size.rowHeightSmall
end

local function AddRow(child, yOffset, label, value, index)
    local T = GearPath.Theme
    local row = CreateFrame("Frame", nil, child)
    row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, yOffset)
    row:SetHeight(T.size.rowHeightSmall)

    if index and index % 2 == 0 then
        local bg = row:CreateTexture(nil, "BACKGROUND")
        bg:SetAllPoints()
        bg:SetColorTexture(unpack(T.color.whiteFaint))
    end

    local labelText = row:CreateFontString(nil, "OVERLAY", T.font.body)
    labelText:SetPoint("LEFT", row, "LEFT", T.space.sm, 0)
    labelText:SetText(label)
    labelText:SetTextColor(unpack(T.color.textMuted))
    labelText:SetWidth(90)

    local valueText = row:CreateFontString(nil, "OVERLAY", T.font.bodyBright)
    valueText:SetPoint("LEFT", labelText, "RIGHT", T.space.xs, 0)
    valueText:SetPoint("RIGHT", row, "RIGHT", -T.space.xs, 0)
    valueText:SetText(value)
    valueText:SetJustifyH("LEFT")

    return yOffset - T.size.rowHeightSmall
end

-- ============================================================
-- Tab lifecycle
-- ============================================================

function StatsTab:Show(parent)
    if not container then
        container = CreateFrame("ScrollFrame", "GearPathStatsTab", parent, "UIPanelScrollFrameTemplate")
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

function StatsTab:Hide()
    if container then container:Hide() end
end

function StatsTab:Refresh()
    if not container or not container:IsShown() then return end

    local T = GearPath.Theme

    local child = container.child
    for _, c in pairs({ child:GetChildren() }) do c:Hide() end
    for _, r in pairs({ child:GetRegions() }) do r:Hide() end

    local guide = GearPath:GetGuideForCurrentSpec()
    if not guide then
        local msg = child:CreateFontString(nil, "OVERLAY", T.font.label)
        msg:SetPoint("TOP", child, "TOP", 0, -40)
        msg:SetWidth(child:GetWidth() - 20)
        msg:SetText("No guide data available for this spec.\nSelect a hero talent in-game to see recommendations.")
        msg:SetTextColor(unpack(T.color.textMuted))
        msg:SetJustifyH("CENTER")
        child:SetHeight(100)
        return
    end

    local y = -T.space.xs

    -- Stat Priority
    y = AddHeader(child, y, "Stat Priority")
    local stats = guide.Stats
    if stats and stats.ordered then
        for i, stat in ipairs(stats.ordered) do
            local row = CreateFrame("Frame", nil, child)
            row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, y)
            row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, y)
            row:SetHeight(T.size.rowHeightSmall)

            if i % 2 == 0 then
                local bg = row:CreateTexture(nil, "BACKGROUND")
                bg:SetAllPoints()
                bg:SetColorTexture(unpack(T.color.whiteFaint))
            end

            local numText = row:CreateFontString(nil, "OVERLAY", T.font.body)
            numText:SetPoint("LEFT", row, "LEFT", T.space.sm, 0)
            numText:SetText(i .. ".")
            numText:SetTextColor(unpack(T.color.textMuted))
            numText:SetWidth(20)

            local statText = row:CreateFontString(nil, "OVERLAY", T.font.bodyBright)
            statText:SetPoint("LEFT", numText, "RIGHT", T.space.xs, 0)
            statText:SetText(stat)

            y = y - T.size.rowHeightSmall
        end
    end
    if stats and stats.note and stats.note ~= "" then
        local note = child:CreateFontString(nil, "OVERLAY", T.font.body)
        note:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.sm, y - 2)
        note:SetWidth(child:GetWidth() - 20)
        note:SetText(stats.note)
        note:SetTextColor(unpack(T.color.textMuted))
        note:SetJustifyH("LEFT")
        note:SetWordWrap(true)
        y = y - (note:GetStringHeight() + 6)
    end

    y = y - 10

    -- Gems
    if guide.Gems and #guide.Gems > 0 then
        y = AddHeader(child, y, "Gems")
        for i, gem in ipairs(guide.Gems) do
            y = AddRow(child, y, "", gem.name, i)
        end
        y = y - 10
    end

    -- Enchants
    if guide.Enchants and #guide.Enchants > 0 then
        y = AddHeader(child, y, "Enchants")
        for i, ench in ipairs(guide.Enchants) do
            y = AddRow(child, y, ench.slot, ench.enchant, i)
        end
        y = y - 10
    end

    -- Consumables
    local cons = guide.Consumables
    if cons then
        local hasAny = (cons.flask and cons.flask ~= "")
            or (cons.food and cons.food ~= "")
            or (cons.potion and cons.potion ~= "")
            or (cons.weaponOil and cons.weaponOil ~= "")
            or (cons.augRune and cons.augRune ~= "")

        if hasAny then
            y = AddHeader(child, y, "Consumables")
            local idx = 0
            local function addCons(label, value)
                if value and value ~= "" then
                    idx = idx + 1
                    y = AddRow(child, y, label, value, idx)
                end
            end
            addCons("Flask", cons.flask)
            addCons("Food", cons.food)
            addCons("Potion", cons.potion)
            addCons("Weapon Oil", cons.weaponOil)
            addCons("Aug. Rune", cons.augRune)
        end
    end

    y = y - 10
    local attribution = child:CreateFontString(nil, "OVERLAY", T.font.body)
    attribution:SetPoint("TOPLEFT", child, "TOPLEFT", T.space.xs, y)
    attribution:SetText("Data sourced from Icy Veins.")
    attribution:SetTextColor(unpack(T.color.textMuted))
    y = y - T.size.rowHeightSmall

    child:SetHeight(math.abs(y) + 10)
end
