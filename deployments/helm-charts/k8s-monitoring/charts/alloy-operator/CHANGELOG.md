# Changelog

## Unreleased

## 0.3.2

* Add the ability to restrict the Alloy Operator to specific namespaces (@petewall)

## 0.3.1

* Update Alloy to 1.1.1 (@petewall)

## 0.3.0

* Removing default resource requests and limits (@petewall, @kespineira)
* Adding values.schema.json to validate the inputs (@petewall)
* Add GitHub Action linting (@petewall)
* Add the ability to enable or disable RBAC object creation (@petewall)

## 0.2.10

* Update Alloy to 1.1.0 (@petewall)

## 0.2.9

* Added more integration tests (@petewall)
* Added RBAC rules for HPA (@petewall)
* Added RBAC rules for Ingress, Prometheus Rules, and Prom operator objects (@discostur)
* Added RBAC rules for PDB (@tw-sematell)

## 0.2.8

* Set node selector by default (@petewall)
* Update the build flag so it updates based on alloy version (@petewall)
* Fix YAML linting (@petewall)
* Set Apache 2.0 license (@tomwilkie)

## 0.2.7

* Update Alloy to 1.0.3 (@petewall)
* Include the CRD in the GitHub release (@petewall)

## 0.2.2

* Added the ability to override image registry and pull secrets with the global option (@petewall)
* Added alloy-values.yaml, the default values, to the Helm chart as a base file (@petewall)

## 0.2.1

* Updated README (@petewall)

## 0.2.0

* First Helm release (@petewall)

## 0.1.0

* Initial prototype (@petewall)
