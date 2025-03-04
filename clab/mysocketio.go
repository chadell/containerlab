// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package clab

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/srl-labs/containerlab/types"
)

var supportedSockTypes = []string{"tcp", "tls", "http", "https"}

type mysocket struct {
	Stype          string
	Port           int
	AllowedDomains []string
	AllowedEmails  []string
}

func parseSocketCfg(s string) (mysocket, error) {
	var err error
	ms := mysocket{}
	split := strings.Split(s, "/")
	if len(split) > 3 {
		return ms, fmt.Errorf("wrong mysocketio publish section %s. should be <type>/<port-number>[/<allowed-domains>|<email>,], i.e. tcp/22 or tls/22/gmail.com or http/80/user1@mail.com,gmail.com,user2@clab.com", s)
	}

	if err = checkSockType(split[0]); err != nil {
		return ms, err
	}
	ms.Stype = split[0]
	p, err := strconv.Atoi(split[1]) // port
	if err != nil {
		return ms, err
	}
	if err := checkSockPort(p); err != nil {
		return ms, err
	}
	ms.Port = p

	if len(split) == 3 {
		ms.AllowedDomains, ms.AllowedEmails, _ = parseAllowedUsers(split[2])

		// identity aware sockets for TCP require TLS type. Force the switch to make it easy on users
		if (len(ms.AllowedDomains) > 0 || len(ms.AllowedEmails) > 0) && ms.Stype == "tcp" {
			ms.Stype = "tls"
		}
	}

	return ms, err
}

func parseAllowedUsers(s string) (domains, emails []string, err error) {

	for _, e := range strings.Split(s, ",") {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.Contains(e, "@") {
			emails = append(emails, e)
		} else {
			domains = append(domains, e)
		}
	}
	return domains, emails, err
}

func checkSockType(t string) error {
	if _, ok := StringInSlice(supportedSockTypes, t); !ok {
		return fmt.Errorf("mysocketio type %s is not supported. Supported types are tcp/tls/http/https", t)
	}
	return nil
}

func checkSockPort(p int) error {
	if p < 1 || p > 65535 {
		return fmt.Errorf("incorrect port number %v", p)
	}
	return nil
}

// createMysocketTunnels creates internet reachable personal tunnels using mysocket.io
func createMysocketTunnels(ctx context.Context, c *CLab, node *types.NodeConfig) error {
	// remove the existing sockets
	cmd := []string{"/bin/sh", "-c", "mysocketctl socket ls | awk '/clab/ {print $2}' | xargs -n1 mysocketctl socket delete -s"}
	log.Debugf("Running postdeploy mysocketio command %q", cmd)
	_, _, err := c.Runtime.Exec(ctx, node.ContainerID, cmd)
	if err != nil {
		return fmt.Errorf("failed to remove existing sockets: %v", err)
	}

	for _, n := range c.Nodes {
		if len(n.Publish) == 0 {
			continue
		}
		for _, socket := range n.Publish {
			ms, err := parseSocketCfg(socket)
			if err != nil {
				return err
			}

			// create socket and get its ID
			sockCmd := createSockCmd(ms, n.ShortName)
			cmd := []string{"/bin/sh", "-c", fmt.Sprintf("%s | awk 'NR==4 {print $2}'", sockCmd)}
			log.Debugf("Running mysocketio command %q", cmd)
			stdout, _, err := c.Runtime.Exec(ctx, node.ContainerID, cmd)
			if err != nil {
				return fmt.Errorf("failed to create mysocketio socket: %v", err)
			}
			sockID := strings.TrimSpace(string(stdout))

			// create tunnel and get its ID
			cmd = []string{"/bin/sh", "-c", fmt.Sprintf("mysocketctl tunnel create -s %s | awk 'NR==4 {print $4}'", sockID)}
			log.Debugf("Running mysocketio command %q", cmd)
			stdout, _, err = c.Runtime.Exec(ctx, node.ContainerID, cmd)
			if err != nil {
				return fmt.Errorf("failed to create mysocketio socket: %v", err)
			}
			tunID := strings.TrimSpace(string(stdout))

			// connect tunnel
			cmd = []string{"/bin/sh", "-c", fmt.Sprintf("mysocketctl tunnel connect --host %s -p %d -s %s -t %s > socket-%s-%s-%d.log",
				n.LongName, ms.Port, sockID, tunID, n.ShortName, ms.Stype, ms.Port)}
			log.Debugf("Running mysocketio command %q", cmd)
			err = c.Runtime.ExecNotWait(ctx, node.ContainerID, cmd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func createSockCmd(ms mysocket, n string) string {
	cmd := fmt.Sprintf("mysocketctl socket create -t %s -n clab-%s-%s-%d", ms.Stype, n, ms.Stype, ms.Port)
	if len(ms.AllowedDomains) > 0 || len(ms.AllowedEmails) > 0 {
		cmd = fmt.Sprintf("%s -c", cmd)
	}
	if len(ms.AllowedDomains) > 0 {
		cmd = fmt.Sprintf("%s -d '%s'", cmd, strings.Join(ms.AllowedDomains, ","))
	}
	if len(ms.AllowedEmails) > 0 {
		cmd = fmt.Sprintf("%s -e '%s'", cmd, strings.Join(ms.AllowedEmails, ","))
	}
	return cmd
}
