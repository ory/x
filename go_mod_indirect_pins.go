// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

//go:build go_mod_indirect_pins
// +build go_mod_indirect_pins

package x

import (
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/gorilla/websocket"
	_ "github.com/jandelgado/gcov2lcov"

	_ "github.com/ory/go-acc"
)
