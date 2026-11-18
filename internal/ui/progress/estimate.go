// SPDX-License-Identifier: GPL-3.0-or-later

package progress

import (
	"fmt"
	"time"
)

const (
	// Prima di questa soglia la stima del tempo rimanente è troppo instabile
	// per essere mostrata.
	minEstimateElapsed  = 3 * time.Second
	minEstimateFraction = 0.01
	// barMax è il valore massimo della barra: in millesimi, per un
	// avanzamento fluido anche su operazioni molto lunghe.
	barMax = 1000
)

// fraction restituisce la frazione completata, compresa tra 0 e 1.
func fraction(done, total int64) float64 {
	if total <= 0 || done <= 0 {
		return 0
	}
	if done >= total {
		return 1
	}
	return float64(done) / float64(total)
}

// estimateRemaining stima il tempo rimanente supponendo una velocità costante.
// Restituisce false se non ci sono ancora dati sufficienti.
func estimateRemaining(elapsed time.Duration, frac float64) (time.Duration, bool) {
	if frac >= 1 {
		return 0, true
	}
	if elapsed < minEstimateElapsed || frac < minEstimateFraction {
		return 0, false
	}
	total := time.Duration(float64(elapsed) / frac)
	return total - elapsed, true
}

// formatRemaining restituisce il tempo rimanente in forma leggibile,
// arrotondato per non far oscillare continuamente le cifre.
func formatRemaining(d time.Duration) string {
	switch {
	case d < 5*time.Second:
		return "pochi secondi"
	case d < time.Minute:
		// Multipli di 5 secondi.
		s := int((d + 4*time.Second) / (5 * time.Second) * 5)
		if s >= 60 {
			return "1 min"
		}
		return fmt.Sprintf("%d s", s)
	case d < time.Hour:
		d = d.Round(10 * time.Second)
		m, s := int(d/time.Minute), int(d%time.Minute/time.Second)
		if s == 0 {
			return fmt.Sprintf("%d min", m)
		}
		return fmt.Sprintf("%d min %d s", m, s)
	default:
		d = d.Round(time.Minute)
		h, m := int(d/time.Hour), int(d%time.Hour/time.Minute)
		if m == 0 {
			return fmt.Sprintf("%d h", h)
		}
		return fmt.Sprintf("%d h %d min", h, m)
	}
}
