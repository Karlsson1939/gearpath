-- GearPath
-- UI/VaultTab.lua - Vault advisor view (M5)

GearPath.VaultTab = {}
local VaultTab = GearPath.VaultTab

local container = nil

function VaultTab:Show(parent)
    if not container then
        container = CreateFrame("Frame", "GearPathVaultTab", parent)
        container:SetAllPoints(parent)

        local label = container:CreateFontString(nil, "OVERLAY", "GameFontNormal")
        label:SetPoint("CENTER", container, "CENTER", 0, 0)
        label:SetText("Vault Advisor coming in M5")
        label:SetTextColor(0.6, 0.6, 0.6)
    else
        container:SetParent(parent)
        container:SetAllPoints(parent)
    end
    container:Show()
end

function VaultTab:Hide()
    if container then container:Hide() end
end