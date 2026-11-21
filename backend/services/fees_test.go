package services

import (
	"testing"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/stretchr/testify/assert"
)

func TestFeeService_CalculateIntegerArithmetic(t *testing.T) {
	cfg := &config.Config{
		PlatformFeeBps:   50, // 0.50%
		ForexFeeBps:      25, // 0.25%
		ComplianceFeeBps: 10, // 0.10%
		NetworkFeeBps:    15, // 0.15%
		MinFee:           0,
		MaxFee:           0,
	}

	svc := NewFeeService(cfg)

	// $100.00 = 10000 cents
	breakdown := svc.Calculate(10000)

	assert.Equal(t, int64(50), breakdown.PlatformFee)
	assert.Equal(t, int64(25), breakdown.ForexFee)
	assert.Equal(t, int64(10), breakdown.ComplianceFee)
	assert.Equal(t, int64(15), breakdown.NetworkFee)
	assert.Equal(t, int64(100), breakdown.TotalFee)
}

func TestFeeService_CalculateWithMinFeeClamp(t *testing.T) {
	cfg := &config.Config{
		PlatformFeeBps:   50,
		ForexFeeBps:      25,
		ComplianceFeeBps: 10,
		NetworkFeeBps:    15,
		MinFee:           200, // min fee 200 cents ($2.00)
		MaxFee:           0,
	}

	svc := NewFeeService(cfg)

	breakdown := svc.Calculate(10000)
	assert.Equal(t, int64(200), breakdown.TotalFee)
}

// TestFeeParityWithContract pins the same vectors as
// contracts/tests/escrow_remittance_flow.rs (FEE_VECTORS). The on-chain
// contracts enforce fees, so the backend quote must match them exactly.
// If you change one side, change the other.
func TestFeeParityWithContract(t *testing.T) {
	vectors := []struct {
		name     string
		amount   int64
		bps      int
		min, max float64 // 0 max == unlimited (contract: i128::MAX)
		expected int64
	}{
		{"plain 2.5%", 10000, 250, 0, 0, 250},
		{"min clamp", 1000, 10, 100, 1000, 100},
		{"max clamp", 1000000, 500, 100, 1000, 1000},
		{"rounds down", 999, 250, 0, 0, 24},
		{"dust rounds to zero", 1, 250, 0, 0, 0},
		{"large amount", 123456789, 175, 0, 0, 2160493},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			svc := NewFeeService(&config.Config{
				PlatformFeeBps: v.bps,
				MinFee:         v.min,
				MaxFee:         v.max,
			})
			assert.Equal(t, v.expected, svc.Calculate(v.amount).TotalFee)
		})
	}
}
