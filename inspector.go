package inspector

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/expanded_tx"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"

	"github.com/pkg/errors"
)

/**
 * Inspector Service
 *
 * What is my purpose?
 * - You look at Bitcoin transactions that I give you
 * - You tell me if they contain return data of interest
 * - You give me back special transaction objects (ITX objects)
 */

var (
	// ErrDecodeFail Failed to decode a transaction payload
	ErrDecodeFail = errors.New("Failed to decode payload")

	// ErrInvalidProtocol The op return data was invalid
	ErrInvalidProtocol = errors.New("Invalid protocol message")

	// ErrMissingInputs
	ErrMissingInputs = errors.New("Message is missing inputs")

	// ErrMissingOutputs
	ErrMissingOutputs = errors.New("Message is missing outputs")

	ErrNegativeFee  = errors.New("Negative fee")
	ErrUnpromotedTx = errors.New("Unpromoted tx")
	ErrIncompleteTx = errors.New("Incomplete tx")

	// prefixP2PKH Pay to PKH prefix
	prefixP2PKH = []byte{0x76, 0xA9}
)

// NodeInterface represents a configured bitcoin node that is capable
// of looking up transactions and parameters for its network
type NodeInterface interface {
	SaveTx(context.Context, *wire.MsgTx) error
	GetTx(context.Context, bitcoin.Hash32) (*wire.MsgTx, error)
	GetOutputs(context.Context, []wire.OutPoint) ([]bitcoin.UTXO, error)
}

// NewTransaction builds an ITX from a raw transaction.
func NewTransaction(ctx context.Context, raw string, isTest bool) (*Transaction, error) {
	data := strings.Trim(string(raw), "\n ")

	b, err := hex.DecodeString(data)
	if err != nil {
		return nil, errors.Wrap(ErrDecodeFail, "decoding string")
	}

	// Set up the Wire transaction
	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)
	if err := tx.Deserialize(buf); err != nil {
		return nil, errors.Wrap(ErrDecodeFail, "deserializing wire message")
	}

	return NewTransactionFromWire(ctx, &tx, isTest)
}

// NewTransactionFromHash builds an ITX from a transaction hash
func NewTransactionFromHash(ctx context.Context, node NodeInterface, hash bitcoin.Hash32,
	isTest bool) (*Transaction, error) {

	tx, err := node.GetTx(ctx, hash)
	if err != nil {
		return nil, err
	}

	return NewTransactionFromHashWire(ctx, hash, tx, isTest)
}

// NewTransactionFromWire builds an ITX from a wire Msg Tx
func NewTransactionFromWire(ctx context.Context, tx *wire.MsgTx,
	isTest bool) (*Transaction, error) {

	return NewTransactionFromHashWire(ctx, *tx.TxHash(), tx, isTest)
}

// NewTransactionFromWire builds an ITX from a wire Msg Tx
func NewTransactionFromHashWire(ctx context.Context, hash bitcoin.Hash32, tx *wire.MsgTx,
	isTest bool) (*Transaction, error) {

	// Must have inputs
	if len(tx.TxIn) == 0 {
		return nil, errors.Wrap(ErrMissingInputs, "parsing transaction")
	}

	// Must have outputs
	if len(tx.TxOut) == 0 {
		return nil, errors.Wrap(ErrMissingOutputs, "parsing transaction")
	}

	result := &Transaction{
		Hash:  hash,
		MsgTx: tx.Copy(),
	}

	if err := result.Setup(ctx, isTest); err != nil {
		return nil, errors.Wrap(err, "setup")
	}

	return result, nil
}

// NewBaseTransactionFromHashWire creates a non-setup transaction from an already calculated tx hash
// and a wire tx. This is the same as NewTransactionFromHashWire except Setup must be called
// before the transaction is usable.
func NewBaseTransactionFromHashWire(ctx context.Context, hash bitcoin.Hash32,
	tx *wire.MsgTx) (*Transaction, error) {
	return &Transaction{
		Hash:  hash,
		MsgTx: tx.Copy(),
	}, nil
}

// NewBaseTransactionFromWire creates a non-setup transaction from a wire tx. This is the same as
// NewTransactionFromWire except Setup must be called before the transaction is usable.
func NewBaseTransactionFromWire(ctx context.Context, tx *wire.MsgTx) (*Transaction, error) {
	return &Transaction{
		Hash:  *tx.TxHash(),
		MsgTx: tx.Copy(),
	}, nil
}

func NewTransactionFromTxBuilder(ctx context.Context, tx *txbuilder.TxBuilder,
	isTest bool) (*Transaction, error) {

	result, err := NewTransactionFromWire(ctx, tx.MsgTx, isTest)
	if err != nil {
		return result, errors.Wrap(err, "new from wire")
	}

	utxos := make([]bitcoin.UTXO, 0, len(tx.Inputs))
	for i, input := range tx.Inputs {
		if tx.MsgTx.TxIn[i].PreviousOutPoint.Index == 0xffffffff {
			continue // skip coinbase inputs
		}
		utxos = append(utxos, bitcoin.UTXO{
			Hash:          tx.MsgTx.TxIn[i].PreviousOutPoint.Hash,
			Index:         tx.MsgTx.TxIn[i].PreviousOutPoint.Index,
			Value:         input.Value,
			LockingScript: input.LockingScript,
		})
	}

	if err := result.PromoteFromUTXOs(ctx, utxos, isTest); err != nil {
		return result, errors.Wrap(err, "promote")
	}

	return result, nil
}

func NewTransactionFromTransactionWithOutputs(ctx context.Context,
	tx expanded_tx.TransactionWithOutputs, isTest bool) (*Transaction, error) {

	result, err := NewTransactionFromWire(ctx, tx.GetMsgTx(), isTest)
	if err != nil {
		return result, errors.Wrap(err, "new from wire")
	}

	inputCount := tx.InputCount()
	utxos := make([]bitcoin.UTXO, 0, inputCount)
	for i := 0; i < inputCount; i++ {
		input := tx.Input(i)
		output, err := tx.InputOutput(i)
		if err != nil {
			return nil, errors.Wrapf(err, "input %d", i)
		}

		if input.PreviousOutPoint.Index == 0xffffffff {
			continue // skip coinbase inputs
		}
		utxos = append(utxos, bitcoin.UTXO{
			Hash:          input.PreviousOutPoint.Hash,
			Index:         input.PreviousOutPoint.Index,
			Value:         output.Value,
			LockingScript: output.LockingScript,
		})
	}

	if err := result.PromoteFromUTXOs(ctx, utxos, isTest); err != nil {
		return result, errors.Wrap(err, "promote")
	}

	return result, nil
}

func NewTransactionFromOutputs(ctx context.Context, hash bitcoin.Hash32, tx *wire.MsgTx,
	outputs []*wire.TxOut, isTest bool) (*Transaction, error) {

	result, err := NewBaseTransactionFromHashWire(ctx, hash, tx)
	if err != nil {
		return nil, errors.Wrap(err, "new")
	}

	if len(tx.TxIn) != len(outputs) {
		return nil, ErrMissingOutputs
	}

	utxos := make([]bitcoin.UTXO, 0, len(tx.TxIn))
	for i, input := range tx.TxIn {
		if input.PreviousOutPoint.Index == 0xffffffff {
			continue // skip coinbase inputs
		}

		utxos = append(utxos, bitcoin.UTXO{
			Hash:          input.PreviousOutPoint.Hash,
			Index:         input.PreviousOutPoint.Index,
			Value:         outputs[i].Value,
			LockingScript: outputs[i].LockingScript,
		})
	}

	if err := result.PromoteFromUTXOs(ctx, utxos, isTest); err != nil {
		return nil, errors.Wrap(err, "promote")
	}

	return result, nil
}
