#!/usr/bin/env bash
set -euo pipefail

GEN_DIR="/var/lib/grafana/dashboards/generated"
ARCHIVES_DIR="/var/lib/grafana/archives"

mkdir -p "${GEN_DIR}"

# Generate a simple dashboard JSON for each CSV file found under ARCHIVES_DIR.
# Each dashboard contains a single time series panel that reads the CSV with Infinity and groups series by the `label` column.

index=0
shopt -s nullglob globstar
for csv in "${ARCHIVES_DIR}"/**/*.csv "${ARCHIVES_DIR}"/*.csv; do
  [ -e "$csv" ] || continue
  rel_path="${csv#${ARCHIVES_DIR}/}"
  name_no_ext="${rel_path%.csv}"
  dash_uid="csv_$(echo -n "$name_no_ext" | tr '/.' '__' | tr -cd '[:alnum:]_-' | cut -c1-40)"
  panel_id=$((100 + index))
  dashboard_json="${GEN_DIR}/${dash_uid}.json"

  cat >"${dashboard_json}" <<JSON
{
  "id": null,
  "uid": "${dash_uid}",
  "title": "CSV: ${name_no_ext}",
  "timezone": "browser",
  "tags": ["csv", "infinity"],
  "schemaVersion": 38,
  "version": 1,
  "refresh": "",
  "time": { "from": "now-30d", "to": "now" },
  "panels": [
    {
      "id": ${panel_id},
      "type": "timeseries",
      "title": "${name_no_ext}",
      "datasource": { "type": "yesoreyeram-infinity-datasource", "uid": "infinity" },
      "targets": [
        {
          "refId": "A",
          "type": "csv",
          "source": "url",
          "format": "timeseries",
          "parser": "backend",
          "url": "http://archives/${rel_path}",
          "url_options":{ "method": "GET" },
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
          "options": {
            "fields": ["label"]
          }
        },
        {
          "id": "renameByRegex",
          "options": {
            "regex": "(time).*",
            "renamePattern": "\$1"
          }
        },
        {
          "id": "renameByRegex",
          "options": {
            "regex": "value(.*)",
            "renamePattern": "\$1"
          }
        }
      ],
      "options": {
        "legend": { "displayMode": "list", "placement": "bottom" },
        "tooltip": { "mode": "multi" }
      },
      "fieldConfig": { "defaults": { "unit": "none" }, "overrides": [] }
    }
  ]
}
JSON
  echo "[generate-dashboards] Generated: ${dashboard_json}"
  index=$((index+1))
done

if [ ${index} -eq 0 ]; then
  echo "[generate-dashboards] No CSV files found under ${ARCHIVES_DIR}."
fi
