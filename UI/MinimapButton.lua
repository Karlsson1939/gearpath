-- GearPath
-- UI/MinimapButton.lua - Minimap button via LibDBIcon

GearPath.MinimapButton = {}
local MinimapButton = GearPath.MinimapButton

function MinimapButton:Initialize()
    local LDB = LibStub("LibDataBroker-1.1")
    local LibDBIcon = LibStub("LibDBIcon-1.0")

    local dataObject = LDB:NewDataObject("GearPath", {
        type  = "launcher",
        icon  = "Interface\\Icons\\inv_misc_gear_01",
        label = "GearPath",

        OnClick = function(_, button)
            if button == "LeftButton" and GearPath.MainFrame then
                GearPath.MainFrame:Toggle()
            end
        end,

        OnTooltipShow = function(tooltip)
            local T = GearPath.Theme

            tooltip:AddLine("GearPath")
            tooltip:AddLine(" ")

            if GearPath.currentClass and GearPath.currentSpec then
                tooltip:AddLine(GearPath.Detection:GetSummary(), unpack(T.color.textPrimary))
            end

            if GearPath.PriorityEngine then
                local top = GearPath.PriorityEngine:GetTop(1)
                if top and top[1] then
                    tooltip:AddLine(" ")
                    tooltip:AddLine("Top priority:", unpack(T.color.accentGold))
                    tooltip:AddLine(
                        top[1].name .. " (" .. top[1].missingCount .. " item(s))",
                        unpack(T.color.textPrimary)
                    )
                else
                    tooltip:AddLine("No missing BiS items!", unpack(T.color.statusEquipped))
                end
            end

            tooltip:AddLine(" ")
            tooltip:AddLine("Left-click to open", unpack(T.color.textMuted))
        end,
    })

    LibDBIcon:Register("GearPath", dataObject, GearPath.db.profile.minimap)
end
