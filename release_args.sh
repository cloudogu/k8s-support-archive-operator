#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

componentTemplateFile=k8s/helm/component-patch-tpl.yaml

# this function will be sourced from release.sh and be called from release_functions.sh
update_versions_modify_files() {
  newReleaseVersion="${1}"
  valuesYAML=k8s/helm/values.yaml
  componentPatchTplYAML=k8s/helm/component-patch-tpl.yaml

  ./.bin/yq -i ".controllerManager.manager.image.tag = \"${newReleaseVersion}\"" "${valuesYAML}"
  ./.bin/yq -i ".values.images.supportArchiveOperator |= sub(\":(([0-9]+)\.([0-9]+)\.([0-9]+)((?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))|(?:\+[0-9A-Za-z-]+))?)\", \":${newReleaseVersion}\")" "${componentPatchTplYAML}"

  local nginxRegistry
  local nginxRepo
  local nginxTag
  nginxRegistry=$(yq '.webserver.image.registry' < "${valuesYAML}")
  nginxRepo=$(yq '.webserver.image.repository' < "${valuesYAML}")
  nginxTag=$(yq '.webserver.image.tag' < "${valuesYAML}")
  setAttributeInComponentPatchTemplate ".values.images.nginxWebserver" "${nginxRegistry}/${nginxRepo}:${nginxTag}"

}

setAttributeInComponentPatchTemplate() {
  local key="${1}"
  local value="${2}"

  yq -i "${key} = \"${value}\"" "${componentTemplateFile}"
}

update_versions_stage_modified_files() {
  valuesYAML=k8s/helm/values.yaml
  componentPatchTplYAML=k8s/helm/component-patch-tpl.yaml

  git add "${valuesYAML}" "${componentPatchTplYAML}"
}
