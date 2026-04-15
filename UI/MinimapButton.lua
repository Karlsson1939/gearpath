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
            if button == "LeftButton" then
                if GearPath.MainFrame then
                    GearPath.MainFrame:Toggle()
                end
            elseif button == "RightButton" then
                if GearPath.Config then
                    GearPath.Config:Open()
                end
            end
        end,

        OnTooltipShow = function(tooltip)
            tooltip:AddLine("GearPath")
            tooltip:AddLine(" ")

            if GearPath.currentClass and GearPath.currentSpec then
                tooltip:AddLine(GearPath.Detection:GetSummary(), 1, 1, 1)
            end

            if GearPath.PriorityEngine then
                local top = GearPath.PriorityEngine:GetTop(1)
                if top and top[1] then
                    tooltip:AddLine(" ")
                    tooltip:AddLine("Top priority:", 0.9, 0.9, 0.3)
                    tooltip:AddLine(
                        top[1].name .. " (" .. top[1].missingCount .. " item(s))",
                        1, 1, 1
                    )
                else
                    tooltip:AddLine("No missing BiS items!", 0.3, 1, 0.3)
                end
            end

            tooltip:AddLine(" ")
            tooltip:AddLine("Left-click to open", 0.7, 0.7, 0.7)
            tooltip:AddLine("Right-click for settings", 0.7, 0.7, 0.7)
        end,
    })

    LibDBIcon:Register("GearPath", dataObject, GearPath.db.profile.minimap)
end