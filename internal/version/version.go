// SPDX-License-Identifier: GPL-3.0-or-later

// Package version espone la versione di Support Tools.
package version

// Version è la versione del programma. Il programma di build la imposta con
// -ldflags "-X" leggendo il file VERSION; nelle build manuali vale "dev".
var Version = "dev"
