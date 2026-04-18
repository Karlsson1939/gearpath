-- GearPath
-- UI/StatsTab.lua - Stats & Consumables reference view

GearPath.StatsTab = {}
local StatsTab = GearPath.StatsTab

local container = nil

-- ============================================================
-- Helpers
-- ============================================================

local function AddHeader(child, yOffset, text)
    local header = child:CreateFontString(nil, "OVERLAY", "GameFontNormal")
    header:SetPoint("TOPLEFT", child, "TOPLEFT", 4, yOffset)
    header:SetText(text)
    header:SetTextColor(1.0, 0.82, 0.0)
    return yOffset - 20
end

local function AddRow(child, yOffset, label, value, index)
    local row = CreateFrame("Frame", nil, child)
    row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, yOffset)
    row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, yOffset)
    row:SetHeight(20)

    if index and index % 2 == 0 then
        local bg = row:CreateTexture(nil, "BACKGROUND")
        bg:SetAllPoints()
        bg:SetColorTexture(1, 1, 1, 0.03)
    end

    local labelText = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
    labelText:SetPoint("LEFT", row, "LEFT", 8, 0)
    labelText:SetText(label)
    labelText:SetTextColor(0.6, 0.6, 0.6)
    labelText:SetWidth(90)

    local valueText = row:CreateFontString(nil, "OVERLAY", "GameFontHighlightSmall")
    valueText:SetPoint("LEFT", labelText, "RIGHT", 4, 0)
    valueText:SetPoint("RIGHT", row, "RIGHT", -4, 0)
    valueText:SetText(value)
    valueText:SetJustifyH("LEFT")

    return yOffset - 20
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

    local child = container.child
    for _, c in pairs({ child:GetChildren() }) do c:Hide() end
    for _, r in pairs({ child:GetRegions() }) do r:Hide() end

    local guide = GearPath:GetGuideForCurrentSpec()
    if not guide then
        local msg = child:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        msg:SetPoint("TOP", child, "TOP", 0, -40)
        msg:SetWidth(child:GetWidth() - 20)
        msg:SetText("No guide data available for this spec.\nSelect a hero talent in-game to see recommendations.")
        msg:SetTextColor(0.5, 0.5, 0.5)
        msg:SetJustifyH("CENTER")
        child:SetHeight(100)
        return
    end

    local y = -4

    -- Stat Priority
    y = AddHeader(child, y, "Stat Priority")
    local stats = guide.Stats
    if stats and stats.ordered then
        for i, stat in ipairs(stats.ordered) do
            local row = CreateFrame("Frame", nil, child)
            row:SetPoint("TOPLEFT", child, "TOPLEFT", 0, y)
            row:SetPoint("TOPRIGHT", child, "TOPRIGHT", 0, y)
            row:SetHeight(20)

            if i % 2 == 0 then
                local bg = row:CreateTexture(nil, "BACKGROUND")
                bg:SetAllPoints()
                bg:SetColorTexture(1, 1, 1, 0.03)
            end

            local numText = row:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
            numText:SetPoint("LEFT", row, "LEFT", 8, 0)
            numText:SetText(i .. ".")
            numText:SetTextColor(0.5, 0.5, 0.5)
            numText:SetWidth(20)

            local statText = row:CreateFontString(nil, "OVERLAY", "GameFontHighlightSmall")
            statText:SetPoint("LEFT", numText, "RIGHT", 4, 0)
            statText:SetText(stat)

            y = y - 20
        end
    end
    if stats and stats.note and stats.note ~= "" then
        local note = child:CreateFontString(nil, "OVERLAY", "GameFontNormalSmall")
        note:SetPoint("TOPLEFT", child, "TOPLEFT", 8, y - 2)
        note:SetWidth(child:GetWidth() - 20)
        note:SetText(stats.note)
        note:SetTextColor(0.6, 0.6, 0.6)
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

    child:SetHeight(math.abs(y) + 10)
end
