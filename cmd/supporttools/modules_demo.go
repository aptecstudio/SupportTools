// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows && demo

package main

// Tool di prova, inclusi solo con il tag "demo" (go run ./build -tags demo).
import (
	_ "github.com/aptecstudio/supporttools/internal/modules/demo"
)
