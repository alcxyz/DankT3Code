import QtQuick
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    pluginId: "dankT3Code"
    popoutWidth: 360
    popoutHeight: 180

    horizontalBarPill: Component {
        StyledText {
            text: "T3 · Dev"
            color: Theme.surfaceText
            font.pixelSize: Theme.fontSizeMedium
        }
    }

    verticalBarPill: Component {
        Column {
            spacing: Theme.spacingXS

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "T3"
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
            }

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Dev"
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeSmall
            }
        }
    }

    popoutContent: Component {
        PopoutComponent {
            headerText: "T3 Code"
            detailsText: "Development scaffold"
            showCloseButton: true

            StyledText {
                width: parent.width
                text: "Live activity is not implemented yet. Follow the v0.1.0 milestone for progress."
                wrapMode: Text.WordWrap
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeMedium
            }
        }
    }
}
