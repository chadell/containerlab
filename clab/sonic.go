// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package clab

import "github.com/srl-labs/containerlab/types"

func initSonicNode(c *CLab, nodeDef *types.NodeDefinition, nodeCfg *types.NodeConfig, user string, envs map[string]string) error {
	var err error

	c.Config.Topology.GetNodeConfig(nodeCfg.ShortName)
	if err != nil {
		return err
	}
	nodeCfg.Image = c.Config.Topology.GetNodeImage(nodeCfg.ShortName)
	nodeCfg.Group = c.Config.Topology.GetNodeGroup(nodeCfg.ShortName)
	nodeCfg.Position = c.Config.Topology.GetNodePosition(nodeCfg.ShortName)
	nodeCfg.User = user

	// rewrite entrypoint so sonic won't start supervisord before we attach veth interfaces
	nodeCfg.Entrypoint = "/bin/bash"

	return err
}
