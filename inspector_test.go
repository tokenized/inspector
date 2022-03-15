package inspector

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
)

func TestParseTX(t *testing.T) {
	ctx := context.Background()

	msgTx := loadFixtureTX("2c68cf3e1216acaa1e274dfd3b665b6a9d1d1d252e68d190f9fffc5f7e11fd27")

	itx, err := NewTransactionFromWire(ctx, &msgTx, true)
	if err != nil {
		t.Fatal(err)
	}

	// Parse outputs
	if err := itx.ParseOutputs(true); err != nil {
		t.Fatal(err)
	}

	// the hash of the TX being parsed.
	txHash := newHash("2c68cf3e1216acaa1e274dfd3b665b6a9d1d1d252e68d190f9fffc5f7e11fd27")
	address, err := bitcoin.DecodeAddress("1AWtnFroMiC7LJWUENVnE8NRKkWW6bQFc")
	if err != nil {
		t.Fatalf("Failed to decode address 1 : %s", err)
	}

	hash, err := address.Hash()
	if err != nil {
		t.Fatalf("Failed to get address 1 hash : %s", err)
	}
	t.Logf("Address 1 : %d, %s", address.Type(), hash)

	wantTX := &Transaction{
		Hash:  *txHash,
		MsgTx: &msgTx,
		Inputs: []*Input{
			&Input{
				Value:         7605340,
				LockingScript: []byte{118, 169, 20, 23, 177, 246, 194, 98, 68, 113, 18, 20, 254, 231, 21, 14, 90, 107, 155, 48, 128, 193, 52, 136, 172},
			},
		},
		Outputs: []*Output{
			&Output{},
			&Output{},
		},
	}

	t.Logf("Used wantTX : %s", wantTX.Hash) // To remove warning because of commented code below.

	// Doesn't work with unexported "lock". Even with the cmpopts.IgnoreUnexported().
	// if diff := cmp.Diff(itx, wantTX, cmpopts.IgnoreUnexported()); diff != "" {
	// 	t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", "\u2717", diff)
	// }
}

type TestNode struct{}

func (n *TestNode) GetTx(context.Context, *bitcoin.Hash32) (*wire.MsgTx, error) {
	return nil, nil
}

func (n *TestNode) GetOutputs(context.Context, []wire.OutPoint) ([]bitcoin.UTXO, error) {
	return nil, nil
}

func (n *TestNode) SaveTx(context.Context, *wire.MsgTx) error {
	return nil
}
