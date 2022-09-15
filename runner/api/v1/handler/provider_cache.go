// Copyright (c) 2015-2022 CloudJ Technology Co., Ltd.

package handler

import (
	"cloudiac/runner"
	"cloudiac/runner/api/ctx"
	"net/http"
	"strings"
)

func RunClearProviderCache(c *ctx.Context) {
	req := runner.RunClearProviderCacheReq{}
	if err := c.BindJSON(&req); err != nil {
		c.Error(err, http.StatusBadRequest)
		return
	}

	count := strings.Count(req.Source, "/")
	if count == 2 {
		ok, _ := runner.DeleteProviderCache("./var/plugin-cache", req.Source, req.Version)
		if ok {
			return
		}
	} else if count == 1 {
		ok, _ := runner.DeleteProviderCache("./var/plugin-cache/registry.terraform.io", req.Source, req.Version)
		if ok {
			return
		}
		ok, _ = runner.DeleteProviderCache("./var/plugin-cache/registry.cloudiac.org", req.Source, req.Version)
		if ok {
			return
		}
		ok, _ = runner.DeleteProviderCache("./var/plugin-cache/iac-registry.idcos.com", req.Source, req.Version)
		if ok {
			return
		}
	}
}
