-- GearPath
-- UI/MainFrame.lua - Primary panel container and tab system

GearPath.MainFrame = {}
local MainFrame = GearPath.MainFrame

local frame = nil
local activeTab = 1

function MainFrame:Create()
    if frame then return end

    local T  = GearPath.Theme
    local db = GearPath.db.profile.ui

    -- Main frame
    frame = CreateFrame("Frame", "GearPathMainFrame", UIParent, "BackdropTemplate")
    frame:SetSize(db.frameWidth or 420, db.frameHeight or 520)
    frame:SetPoint(
        db.framePoint.point or "CENTER",
        UIParent,
        db.framePoint.point or "CENTER",
        db.framePoint.x or 0,
        db.framePoint.y or 0
    )
    frame:SetFrameStrata("MEDIUM")
    frame:SetMovable(true)
    frame:SetClampedToScreen(true)
    frame:EnableMouse(true)
    frame:RegisterForDrag("LeftButton")
    frame:SetScript("OnDragStart", function(f) f:StartMoving() end)
    frame:SetScript("OnDragStop", function(f)
        f:StopMovingOrSizing()
        local point, _, _, x, y = f:GetPoint()
        GearPath.db.profile.ui.framePoint = { point = point, x = x, y = y }
    end)

    frame:SetBackdrop({
        bgFile   = "Interface\\DialogFrame\\UI-DialogBox-Background-Dark",
        edgeFile = T.border.frame.edgeFile,
        tile     = true, tileSize = 32, edgeSize = T.border.frame.edgeSize,
        insets   = { left = T.border.frame.insets[1], right = T.border.frame.insets[2],
                     top = T.border.frame.insets[3], bottom = T.border.frame.insets[4] },
    })
    frame:SetBackdropBorderColor(unpack(T.color.classAccent))

    table.insert(UISpecialFrames, "GearPathMainFrame")

    -- Title
    local title = frame:CreateFontString(nil, "OVERLAY", T.font.title)
    title:SetPoint("TOPLEFT", frame, "TOPLEFT", T.space.lg, -14)
    title:SetText("GearPath")

    -- Spec label
    frame.specLabel = frame:CreateFontString(nil, "OVERLAY", T.font.body)
    frame.specLabel:SetPoint("LEFT", title, "RIGHT", T.space.sm, 0)
    frame.specLabel:SetTextColor(unpack(T.color.classAccent))
    frame.specLabel:SetText("")

    -- Close button
    local closeBtn = CreateFrame("Button", nil, frame, "UIPanelCloseButton")
    closeBtn:SetPoint("TOPRIGHT", frame, "TOPRIGHT", 2, 2)
    closeBtn:SetScript("OnClick", function() frame:Hide() end)

    -- Divider line under title
    local divider = frame:CreateTexture(nil, "ARTWORK")
    divider:SetColorTexture(unpack(T.color.whiteDivider))
    divider:SetPoint("TOPLEFT", frame, "TOPLEFT", T.space.md, -34)
    divider:SetPoint("TOPRIGHT", frame, "TOPRIGHT", -T.space.md, -34)
    divider:SetHeight(1)

    -- Tab buttons
    frame.tabs = {}
    local tabLabels = { "Priority", "BiS List", "Stats", "Vault" }
    local tabDescriptions = {
        "Ranked content sources for your missing BiS gear",
        "Best in Slot items with equipped/upgrade status",
        "Stat priorities, gems, enchants, and consumables",
        "Weekly Great Vault progress and recommendations",
    }
    for i, label in ipairs(tabLabels) do
        local tab = CreateFrame("Button", "GearPathTab" .. i, frame)
        tab:SetSize(T.size.tabButtonW, T.size.tabButtonH)
        tab:SetID(i)

        if i == 1 then
            tab:SetPoint("TOPLEFT", frame, "TOPLEFT", T.space.md, -38)
        else
            tab:SetPoint("LEFT", frame.tabs[i-1], "RIGHT", 2, 0)
        end

        local tabBg = tab:CreateTexture(nil, "BACKGROUND")
        tabBg:SetAllPoints()
        tabBg:SetColorTexture(0, 0, 0, 0)
        tab.bg = tabBg

        local tabText = tab:CreateFontString(nil, "OVERLAY", T.font.body)
        tabText:SetAllPoints()
        tabText:SetText(label)
        tab.text = tabText

        tab:SetScript("OnClick", function(btn)
            MainFrame:ShowTab(btn:GetID())
        end)

        tab:SetScript("OnEnter", function(btn)
            if btn:GetID() ~= activeTab then
                btn.bg:SetColorTexture(unpack(T.color.whiteSoft))
            end
            GameTooltip:SetOwner(btn, "ANCHOR_BOTTOM")
            GameTooltip:AddLine(tabDescriptions[btn:GetID()], 1, 1, 1)
            GameTooltip:Show()
        end)

        tab:SetScript("OnLeave", function(btn)
            if btn:GetID() ~= activeTab then
                btn.bg:SetColorTexture(0, 0, 0, 0)
            end
            GameTooltip:Hide()
        end)

        frame.tabs[i] = tab
    end

    -- Tab divider
    local tabDivider = frame:CreateTexture(nil, "ARTWORK")
    tabDivider:SetColorTexture(unpack(T.color.whiteDivider))
    tabDivider:SetPoint("TOPLEFT", frame, "TOPLEFT", T.space.md, -64)
    tabDivider:SetPoint("TOPRIGHT", frame, "TOPRIGHT", -T.space.md, -64)
    tabDivider:SetHeight(1)

    -- Content area
    frame.content = CreateFrame("Frame", "GearPathContent", frame)
    frame.content:SetPoint("TOPLEFT", frame, "TOPLEFT", T.space.md, -70)
    frame.content:SetPoint("BOTTOMRIGHT", frame, "BOTTOMRIGHT", -T.space.md, T.space.md)

    frame:Hide()
    self:ShowTab(GearPath.db.profile.ui.activeTab or 1)
end

function MainFrame:ShowTab(tabIndex)
    if not frame then return end
    local T = GearPath.Theme
    activeTab = tabIndex
    GearPath.db.profile.ui.activeTab = tabIndex

    for i, tab in ipairs(frame.tabs) do
        if i == tabIndex then
            tab.text:SetTextColor(unpack(T.color.classAccent))
            tab.bg:SetColorTexture(unpack(T.color.classAccentSoft))
        else
            tab.text:SetTextColor(unpack(T.color.textMuted))
            tab.bg:SetColorTexture(0, 0, 0, 0)
        end
    end

    -- Hide all tab content first
    if GearPath.PriorityTab then GearPath.PriorityTab:Hide() end
    if GearPath.BiSTab      then GearPath.BiSTab:Hide()      end
    if GearPath.StatsTab    then GearPath.StatsTab:Hide()    end
    if GearPath.VaultTab    then GearPath.VaultTab:Hide()    end

    -- Show active tab
    if tabIndex == 1 and GearPath.PriorityTab then
        GearPath.PriorityTab:Show(frame.content)
    elseif tabIndex == 2 and GearPath.BiSTab then
        GearPath.BiSTab:Show(frame.content)
    elseif tabIndex == 3 and GearPath.StatsTab then
        GearPath.StatsTab:Show(frame.content)
    elseif tabIndex == 4 and GearPath.VaultTab then
        GearPath.VaultTab:Show(frame.content)
    end
end

function MainFrame:Toggle()
    if not frame then self:Create() end
    if frame:IsShown() then
        frame:Hide()
    else
        frame:Show()
        self:Refresh()
    end
end

function MainFrame:IsShown()
    return frame and frame:IsShown()
end

function MainFrame:Refresh()
    if not frame or not frame:IsShown() then return end
    if frame.specLabel then
        frame.specLabel:SetText(GearPath.Detection:GetSummary())
    end
    self:ShowTab(activeTab)
end

function MainFrame:ResetPosition()
    if not frame then return end
    frame:ClearAllPoints()
    frame:SetPoint("CENTER", UIParent, "CENTER", 0, 0)
    GearPath.db.profile.ui.framePoint = { point = "CENTER", x = 0, y = 0 }
end

function MainFrame:GetContentFrame()
    return frame and frame.content
end
