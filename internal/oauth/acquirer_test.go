// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoggerFallsBackToANoOp checks logging cannot crash a sign-in.
func TestLoggerFallsBackToANoOp(t *testing.T) {
	t.Parallel()

	var nilAcquirer *ConfigAcquirer

	for _, acq := range []*ConfigAcquirer{nilAcquirer, {}} {
		log := acq.logger()
		assert.Equal(t, nopLogger{}, log)

		assert.NotPanics(t, func() {
			ctx := t.Context()
			log.Debug(ctx, "m")
			log.Info(ctx, "m")
			log.Warn(ctx, "m")
			log.Error(ctx, "m")
		})
	}
}
