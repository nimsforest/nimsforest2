// Package trees provides tree implementations and registration.
package trees

import (
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/pkg/runtime"
)

func init() {
	// Register PaymentTree
	runtime.RegisterTree(
		"payment",
		"Parses Stripe payment webhooks into payment.completed/payment.failed leaves",
		[]string{"river.stripe.webhook"},
		func(wind *core.Wind, river *core.River) core.Tree {
			return NewPaymentTree(wind, river)
		},
	)

	// Register GeneralTree
	runtime.RegisterTree(
		"general",
		"Parses general events from river.general.> into typed leaves",
		[]string{"river.general.>"},
		func(wind *core.Wind, river *core.River) core.Tree {
			return NewGeneralTree(wind, river)
		},
	)
}
