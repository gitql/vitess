// Copyright 2015, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/gitql/vitess/go/vt/servenv"
	"github.com/gitql/vitess/go/vt/worker/grpcvtworkerserver"
)

func init() {
	servenv.RegisterGRPCFlags()
	servenv.OnRun(func() {
		if servenv.GRPCCheckServiceMap("vtworker") {
			grpcvtworkerserver.StartServer(servenv.GRPCServer, wi)
		}
	})
}
