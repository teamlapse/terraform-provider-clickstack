// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources_test

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/teamlapse/terraform-provider-clickstack/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"clickstack": providerserver.NewProtocol6WithError(provider.New("test")()),
}
