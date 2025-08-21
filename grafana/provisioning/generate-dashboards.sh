#!/usr/bin/env bash
set -euo pipefail

GEN_DIR="/var/lib/grafana/dashboards/generated"
ARCHIVES_DIR="/var/lib/grafana/archives"

mkdir -p "${GEN_DIR}"

# Create one dashboard per subfolder under ARCHIVES_DIR, containing
# one panel per CSV metric file in that subfolder. CSVs directly in ARCHIVES_DIR
# are grouped into a single "root" dashboard.

shopt -s nullglob globstar

# Collect CSV files
csv_files=("${ARCHIVES_DIR}"/**/*.csv "${ARCHIVES_DIR}"/*.csv)

if [ ${#csv_files[@]} -eq 0 ]; then
  echo "[generate-dashboards] No CSV files found under ${ARCHIVES_DIR}."
  exit 0
fi

# Build a list of unique folders relative to ARCHIVES_DIR
folders=()
declare -A seen
for csv in "${csv_files[@]}"; do
  rel_path="${csv#${ARCHIVES_DIR}/}"
  dir_name="$(dirname "${rel_path}")"
  if [ "${dir_name}" = "." ]; then
    dir_name=""  # root
  fi
  if [ -z "${seen["$dir_name"]+x}" ]; then
    folders+=("$dir_name")
    seen["$dir_name"]=1
  fi
done

# Helper to sanitize strings for UID/file names
sanitize() {
  tr '/.' '__' | tr -cd '[:alnum:]_-' | cut -c1-40
}

# Generate one dashboard per folder
for folder in "${folders[@]}"; do
  panels=""
  panel_index=0

  # Dashboard identity
  folder_label="$folder"
  if [ -z "$folder_label" ]; then
    folder_label="root"
  fi
  dash_uid="csv_group_$(echo -n "$folder_label" | sanitize)"
  dashboard_json="${GEN_DIR}/${dash_uid}.json"

  # Build panels for this folder
  for csv in "${csv_files[@]}"; do
    rel_path="${csv#${ARCHIVES_DIR}/}"
    dir_name="$(dirname "${rel_path}")"
    if [ "${dir_name}" = "." ]; then dir_name=""; fi
    if [ "$dir_name" != "$folder" ]; then
      continue
    fi

    name_no_ext="$(basename "${rel_path}")"
    name_no_ext="${name_no_ext%.csv}"

    panel_id=$((100 + panel_index))

    # Create JSON for this panel
    read -r -d '' panel_json <<PANEL || true
    {
      "id": ${panel_id},
      "type": "timeseries",
      "title": "${name_no_ext}",
      "datasource": { "type": "yesoreyeram-infinity-datasource", "uid": "infinity" },
      "gridPos": {
        "x": $((($panel_index % 2) * 12)),
        "y": $(($panel_index / 2)),
        "w": 12,
        "h": 12
      },
      "targets": [
        {
          "refId": "A",
          "type": "csv",
          "source": "url",
          "format": "timeseries",
          "parser": "backend",
          "url": "http://archives/${rel_path}",
          "url_options": { "method": "GET" },
          "csv_options": { "delimiter": ",", "header": true }
        }
      ],
      "transformations": [
        {
          "id": "convertFieldType",
          "options": {
            "conversions": [
              { "targetField": "time", "destinationType": "time" },
              { "targetField": "value", "destinationType": "number" }
            ]
          }
        },
        {
          "id": "partitionByValues",
          "options": { "fields": ["label"] }
        },
        {
          "id": "renameByRegex",
          "options": { "regex": "(time).*", "renamePattern": "\$1" }
        },
        {
          "id": "renameByRegex",
          "options": { "regex": "value(.*)", "renamePattern": "\$1" }
        }
      ],
      "options": {
        "legend": { "displayMode": "list", "placement": "bottom" },
        "tooltip": { "mode": "multi" }
      },
      "fieldConfig": { "defaults": { "unit": "none" }, "overrides": [] }
    }
PANEL

    if [ -n "$panels" ]; then
      panels+=","  # add comma separator
    fi
    panels+="$panel_json"
    panel_index=$((panel_index+1))
  done

  # If folder has no panels (shouldn't happen), skip
  if [ $panel_index -eq 0 ]; then
    echo "[generate-dashboards] Skipping empty folder: ${folder_label}"
    continue
  fi

  # Write dashboard JSON
  cat >"${dashboard_json}" <<JSON
{
  "id": null,
  "uid": "${dash_uid}",
  "title": "CSV: ${folder_label}",
  "timezone": "browser",
  "tags": ["csv", "infinity"],
  "schemaVersion": 38,
  "version": 1,
  "refresh": "",
  "time": { "from": "now-5d", "to": "now" },
  "panels": [
${panels}
  ]
}
JSON
  echo "[generate-dashboards] Generated: ${dashboard_json} (${panel_index} panels)"
done
