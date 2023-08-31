// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !linux

package cgroupslib

import (
	"github.com/hashicorp/nomad/client/lib/idset"
	"github.com/hashicorp/nomad/client/lib/numalib/hwids"
)

// GetPartition creates a no-op Partition that does not do anything.
func GetPartition(*idset.Set[hwids.CoreID]) Partition {
	return NoopPartition()
}
