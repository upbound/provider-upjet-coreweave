// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package templates

import _ "embed"

// ControllerTemplate is the controller.go.tmpl source that upjet renders for
// every generated MR controller. It extends upjet's default with the
// ProviderConfig ReconciliationPolicy wiring.
//
//go:embed controller.go.tmpl
var ControllerTemplate string
