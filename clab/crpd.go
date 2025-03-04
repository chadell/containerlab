// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package clab

import (
	"fmt"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/containerlab/types"
	"github.com/srl-labs/containerlab/utils"
)

func (c *CLab) initCrpdNode(nodeCfg *types.NodeConfig) error {
	var err error

	nodeCfg.Config, err = c.Config.Topology.GetNodeConfig(nodeCfg.ShortName)
	if err != nil {
		return err
	}
	if nodeCfg.Config == "" {
		nodeCfg.Config = defaultConfigTemplates[nodeCfg.Kind]
	}
	// initialize license file
	nodeCfg.License, err = c.Config.Topology.GetNodeLicense(nodeCfg.ShortName)
	if err != nil {
		return err
	}

	// mount config and log dirs
	nodeCfg.Binds = append(nodeCfg.Binds, fmt.Sprint(path.Join(nodeCfg.LabDir, "config"), ":/config"))
	nodeCfg.Binds = append(nodeCfg.Binds, fmt.Sprint(path.Join(nodeCfg.LabDir, "log"), ":/var/log"))
	// mount sshd_config
	nodeCfg.Binds = append(nodeCfg.Binds, fmt.Sprint(path.Join(nodeCfg.LabDir, "config/sshd_config"), ":/etc/ssh/sshd_config"))
	return nil
}

func (c *CLab) createCRPDFiles(nodeCfg *types.NodeConfig) error {
	// create config and logs directory that will be bind mounted to crpd
	utils.CreateDirectory(path.Join(nodeCfg.LabDir, "config"), 0777)
	utils.CreateDirectory(path.Join(nodeCfg.LabDir, "log"), 0777)

	// copy crpd config from default template or user-provided conf file
	cfg := path.Join(nodeCfg.LabDir, "/config/juniper.conf")

	err := nodeCfg.GenerateConfig(cfg, defaultConfigTemplates[nodeCfg.Kind])
	if err != nil {
		log.Errorf("node=%s, failed to generate config: %v", nodeCfg.ShortName, err)
	}

	// copy crpd sshd conf file to crpd node dir
	src := "/etc/containerlab/templates/crpd/sshd_config"
	dst := path.Join(nodeCfg.LabDir, "/config/sshd_config")
	err = utils.CopyFile(src, dst)
	if err != nil {
		return fmt.Errorf("file copy [src %s -> dst %s] failed %v", src, dst, err)
	}
	log.Debugf("CopyFile src %s -> dst %s succeeded\n", src, dst)

	if nodeCfg.License != "" {
		// copy license file to node specific lab directory
		src = nodeCfg.License
		dst = path.Join(nodeCfg.LabDir, "/config/license.conf")
		if err = utils.CopyFile(src, dst); err != nil {
			return fmt.Errorf("file copy [src %s -> dst %s] failed %v", src, dst, err)
		}
		log.Debugf("CopyFile src %s -> dst %s succeeded", src, dst)
	}
	return nil
}
