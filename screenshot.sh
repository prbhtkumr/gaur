#!/bin/bash

# Path to main config and themes directory
MAIN_CONFIG="$HOME/.config/foot/foot.ini"
THEME_DIR="$HOME/.config/foot/themes"
TEMP_CONFIG="/tmp/gaur_theme_preview.ini"

# Map Gaur themes to Foot files
declare -A THEMES=(
    ["Catppuccin Frappe"]="catppuccin-frappe"
    ["Catppuccin Latte"]="catppuccin-latte"
    ["Catppuccin Macchiato"]="catppuccin-macchiato"
    ["Catppuccin Mocha"]="catppuccin-mocha"
    ["Dracula"]="dracula"
    ["Gruvbox Dark"]="gruvbox-dark"
    ["Gruvbox Light"]="gruvbox-light"
    ["Monokai Pro"]="monokai-pro"
    ["One Dark"]="onedark"
    ["Rose Pine"]="rose-pine"
    ["Solarized Dark"]="solarized-dark"
    ["Solarized Light"]="solarized-light"
    ["Tokyonight Day"]="tokyonight-day"
    ["Tokyonight Night"]="tokyonight-night"
    ["Tokyonight Storm"]="tokyonight-storm"
)

# List of themes that need Dark Text (The "Light Themes")
LIGHT_THEMES=("Catppuccin Latte" "Gruvbox Light" "Solarized Light" "Tokyonight Day")

# Helper function to check if a theme is in the light list
is_light_theme() {
    local e match="$1"
    for e in "${LIGHT_THEMES[@]}"; do [[ "$e" == "$match" ]] && return 0; done
    return 1
}

for gaur_theme in "${!THEMES[@]}"; do
    foot_theme="${THEMES[$gaur_theme]}"

    echo "Preparing: $gaur_theme (using $foot_theme)..."

    # Load Main Config
    echo "include=$MAIN_CONFIG" > "$TEMP_CONFIG"

    # Load Theme Config
    echo "include=$THEME_DIR/$foot_theme" >> "$TEMP_CONFIG"

    # FIX - Force Dark Font for Light Themes
    # We append a [colors] section at the very end to override everything else
    if is_light_theme "$gaur_theme"; then
        echo "" >> "$TEMP_CONFIG"
        echo "[colors]" >> "$TEMP_CONFIG"
        echo "foreground=1a1a1a" >> "$TEMP_CONFIG"  # Dark Grey/Black
    fi

    # FIX - Force Window Size in Config (Prevents Segfault)
    echo "" >> "$TEMP_CONFIG"
    echo "[main]" >> "$TEMP_CONFIG"
    echo "initial-window-size-pixels=910x630" >> "$TEMP_CONFIG"

    while true; do
        # We run sh -c inside foot. 
        # If user types n/no, we exit 1. If enter, exit 0.
        foot -c "$TEMP_CONFIG" \
             --title "Screenshot: $gaur_theme" \
             sh -c "
                ./gaur --theme '$gaur_theme'; 
                echo -ne '\n📸 Done? (Enter=Next, n=Retry): '; 
                read -r ans; 
                case \"\$ans\" in 
                    n|N|no|No) exit 1 ;; 
                    *) exit 0 ;; 
                esac"
        
        # Check the exit code of the foot command above
        if [ $? -eq 0 ]; then
            break # User hit Enter (Success), break retry loop, go to next theme
        else
            echo "♻️ Retrying $gaur_theme..."
            # The loop repeats, relaunching the window
        fi
    done
done

echo "All themes previewed!"
