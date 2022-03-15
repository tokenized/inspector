package inspector

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

var (
	// Protocol Request message types
	requestMessageTypes = map[string]bool{
		actions.CodeContractOffer:            true,
		actions.CodeContractAmendment:        true,
		actions.CodeBodyOfAgreementOffer:     true,
		actions.CodeBodyOfAgreementAmendment: true,
		actions.CodeInstrumentDefinition:     true,
		actions.CodeInstrumentModification:   true,
		actions.CodeAssetDefinition:          true, // Deprecated backwards compatibility
		actions.CodeAssetModification:        true, // Deprecated backwards compatibility
		actions.CodeTransfer:                 true,
		actions.CodeProposal:                 true,
		actions.CodeBallotCast:               true,
		actions.CodeOrder:                    true,
		actions.CodeContractAddressChange:    true,
	}

	// Protocol response message types
	responseMessageTypes = map[string]bool{
		actions.CodeInstrumentCreation:       true,
		actions.CodeAssetCreation:            true, // Deprecated backwards compatibility
		actions.CodeContractFormation:        true,
		actions.CodeBodyOfAgreementFormation: true,
		actions.CodeSettlement:               true,
		actions.CodeVote:                     true,
		actions.CodeBallotCounted:            true,
		actions.CodeResult:                   true,
		actions.CodeFreeze:                   true,
		actions.CodeThaw:                     true,
		actions.CodeConfiscation:             true,
		actions.CodeReconciliation:           true,
		actions.CodeRejection:                true,
	}
)

// Transaction represents an ITX (Inspector Transaction) containing
// information about a transaction that is useful to the protocol.
type Transaction struct {
	Hash       bitcoin.Hash32
	MsgTx      *wire.MsgTx
	Inputs     []*Input
	Outputs    []*Output
	RejectCode uint32
	RejectText string

	lock sync.RWMutex
}

func (itx *Transaction) String(net bitcoin.Network) string {
	result := fmt.Sprintf("TxId: %s (%d bytes)\n", itx.Hash, itx.MsgTx.SerializeSize())
	result += fmt.Sprintf("  Version: %d\n", itx.MsgTx.Version)

	result += "  Inputs:\n\n"
	for i, input := range itx.MsgTx.TxIn {
		result += fmt.Sprintf("    Outpoint: %s:%d\n", input.PreviousOutPoint.Hash,
			input.PreviousOutPoint.Index)
		result += fmt.Sprintf("    UnlockingScript: %s\n", input.UnlockingScript)
		result += fmt.Sprintf("    Sequence: %x\n", input.Sequence)

		result += fmt.Sprintf("    LockingScript: %s\n", itx.Inputs[i].LockingScript)
		if !bitcoin.LockingScriptIsUnspendable(itx.Inputs[i].LockingScript) {
			ra, err := bitcoin.RawAddressFromLockingScript(itx.Inputs[i].LockingScript)
			if err == nil {
				result += fmt.Sprintf("    Address: %s\n",
					bitcoin.NewAddressFromRawAddress(ra, net))
			}
		}
		result += fmt.Sprintf("    Value: %d\n", itx.Inputs[i].Value)

		if itx.Inputs[i].Action != nil {
			actionJS, err := json.MarshalIndent(itx.Inputs[i].Action, "      ", "  ")
			if err == nil {
				result += fmt.Sprintf("    Action: \n      %s\n", actionJS)
			}
		} else {
			result += "\n"
		}
	}

	result += "  Outputs:\n\n"
	for i, output := range itx.MsgTx.TxOut {
		result += fmt.Sprintf("    Value: %.08f\n", float32(output.Value)/100000000.0)
		result += fmt.Sprintf("    LockingScript: %s\n", output.LockingScript)
		if !bitcoin.LockingScriptIsUnspendable(output.LockingScript) {
			ra, err := bitcoin.RawAddressFromLockingScript(output.LockingScript)
			if err == nil {
				result += fmt.Sprintf("    Address: %s\n",
					bitcoin.NewAddressFromRawAddress(ra, net))
			}
		}

		if itx.Outputs[i].Action != nil {
			actionJS, err := json.MarshalIndent(itx.Outputs[i].Action, "      ", "  ")
			if err == nil {
				result += fmt.Sprintf("    Action: \n      %s\n", actionJS)
			}
		} else {
			result += "\n"
		}
	}

	result += fmt.Sprintf("  LockTime: %d\n", itx.MsgTx.LockTime)
	return result
}

// Setup finds the tokenized messages.
func (itx *Transaction) Setup(ctx context.Context, isTest bool) error {
	itx.lock.Lock()
	defer itx.lock.Unlock()

	for i, input := range itx.Inputs {
		action, err := protocol.Deserialize(input.LockingScript, isTest)
		if err == nil {
			itx.Inputs[i].Action = action
		}
	}

	if err := itx.ParseOutputs(isTest); err != nil {
		itx.lock.Unlock()
		return errors.Wrap(err, "parse outputs")
	}

	return nil
}

// Validate checks the validity of the data in the protocol message.
func (itx *Transaction) Validate(ctx context.Context) error {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	for _, input := range itx.Inputs {
		if input.Action == nil {
			continue
		}

		if err := input.Action.Validate(); err != nil {
			logger.Warn(ctx, "Protocol message is invalid : %s", err)
			itx.RejectCode = actions.RejectionsMsgMalformed
			itx.RejectText = err.Error()
			return nil
		}
	}

	for _, output := range itx.Outputs {
		if output.Action == nil {
			continue
		}

		if err := output.Action.Validate(); err != nil {
			logger.Warn(ctx, "Protocol message is invalid : %s", err)
			itx.RejectCode = actions.RejectionsMsgMalformed
			itx.RejectText = err.Error()
			return nil
		}
	}

	return nil
}

// PromoteFromUTXOs will populate the inputs and outputs accordingly using UTXOs instead of a node.
func (itx *Transaction) PromoteFromUTXOs(ctx context.Context, utxos []bitcoin.UTXO,
	isTest bool) error {
	itx.lock.Lock()

	if err := itx.ParseInputsFromUTXOs(ctx, utxos, isTest); err != nil {
		itx.lock.Unlock()
		return errors.Wrap(err, "parse inputs")
	}

	if err := itx.ParseOutputs(isTest); err != nil {
		itx.lock.Unlock()
		return errors.Wrap(err, "parse outputs")
	}

	itx.lock.Unlock()
	itx.lock.RLock()
	return nil
}

// Promote will populate the inputs and outputs accordingly
func (itx *Transaction) Promote(ctx context.Context, node NodeInterface, isTest bool) error {
	itx.lock.Lock()

	if err := itx.ParseInputs(ctx, node, isTest); err != nil {
		itx.lock.Unlock()
		return errors.Wrap(err, "parse inputs")
	}

	if err := itx.ParseOutputs(isTest); err != nil {
		itx.lock.Unlock()
		return errors.Wrap(err, "parse outputs")
	}

	itx.lock.Unlock()
	itx.lock.RLock()
	return nil
}

// IsPromoted returns true if inputs and outputs are populated.
func (itx *Transaction) IsPromoted(ctx context.Context) bool {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	return len(itx.Inputs) > 0 && len(itx.Outputs) > 0
}

// ParseInputsFromUTXOs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputsFromUTXOs(ctx context.Context, utxos []bitcoin.UTXO,
	isTest bool) error {

	// Build inputs
	inputs := make([]*Input, len(itx.MsgTx.TxIn))
	offset := 0
	for i, txin := range itx.MsgTx.TxIn {
		if txin.PreviousOutPoint.Index == 0xffffffff {
			// Empty coinbase input
			inputs = append(inputs, &Input{})
			continue
		}

		if !txin.PreviousOutPoint.Hash.Equal(&utxos[offset].Hash) ||
			txin.PreviousOutPoint.Index != utxos[offset].Index {
			return errors.New("Mismatched UTXO")
		}

		inputs[i] = &Input{
			Value:         utxos[offset].Value,
			LockingScript: utxos[offset].LockingScript,
		}

		action, err := protocol.Deserialize(utxos[offset].LockingScript, isTest)
		if err == nil {
			inputs[i].Action = action
		}

		offset++
	}

	itx.Inputs = inputs
	return nil
}

// ParseOutputs sets the Outputs property of the Transaction
func (itx *Transaction) ParseOutputs(isTest bool) error {
	outputs := make([]*Output, len(itx.MsgTx.TxOut))
	for i, txout := range itx.MsgTx.TxOut {
		outputs[i] = &Output{}

		action, err := protocol.Deserialize(txout.LockingScript, isTest)
		if err == nil {
			outputs[i].Action = action
		}
	}

	itx.Outputs = outputs
	return nil
}

// ParseInputs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputs(ctx context.Context, node NodeInterface, isTest bool) error {

	// Fetch input transactions from RPC
	outpoints := make([]wire.OutPoint, 0, len(itx.MsgTx.TxIn))
	for _, txin := range itx.MsgTx.TxIn {
		if txin.PreviousOutPoint.Index != 0xffffffff {
			outpoints = append(outpoints, txin.PreviousOutPoint)
		}
	}

	utxos, err := node.GetOutputs(ctx, outpoints)
	if err != nil {
		return err
	}

	return itx.ParseInputsFromUTXOs(ctx, utxos, isTest)
}

// Returns all the input hashes
func (itx *Transaction) InputHashes() []bitcoin.Hash32 {
	hashes := []bitcoin.Hash32{}

	for _, txin := range itx.MsgTx.TxIn {
		hashes = append(hashes, txin.PreviousOutPoint.Hash)
	}

	return hashes
}

// IsTokenized determines if the inspected transaction is using the Tokenized protocol.
func (itx *Transaction) IsTokenized() bool {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	for _, input := range itx.Inputs {
		if input.Action != nil {
			return true
		}
	}

	for _, output := range itx.Outputs {
		if output.Action != nil {
			return true
		}
	}

	return false
}

// IsRequest returns true if this tx contains a request Tokenized action.
func (itx *Transaction) IsRequest() bool {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	for _, input := range itx.Inputs {
		if input.Action == nil {
			continue
		}

		_, ok := requestMessageTypes[input.Action.Code()]
		if ok {
			return true
		}
	}

	for _, output := range itx.Outputs {
		if output.Action == nil {
			continue
		}

		_, ok := requestMessageTypes[output.Action.Code()]
		if ok {
			return true
		}
	}

	return false
}

// IsResponse returns true if this tx contains a response Tokenized action.
func (itx *Transaction) IsResponse() bool {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	for _, input := range itx.Inputs {
		if input.Action == nil {
			continue
		}

		_, ok := responseMessageTypes[input.Action.Code()]
		if ok {
			return true
		}
	}

	for _, output := range itx.Outputs {
		if output.Action == nil {
			continue
		}

		_, ok := responseMessageTypes[output.Action.Code()]
		if ok {
			return true
		}
	}

	return false
}

func (itx *Transaction) fee() (uint64, error) {
	result := uint64(0)

	if len(itx.Inputs) != len(itx.MsgTx.TxIn) {
		return 0, ErrUnpromotedTx
	}

	for _, input := range itx.Inputs {
		result += input.Value
	}

	for _, output := range itx.MsgTx.TxOut {
		if output.Value > result {
			return 0, ErrNegativeFee
		}
		result -= output.Value
	}

	return result, nil
}

func (itx *Transaction) Fee() (uint64, error) {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	return itx.fee()
}

func (itx *Transaction) FeeRate() (float32, error) {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	fee, err := itx.fee()
	if err != nil {
		return 0.0, err
	}

	size := itx.MsgTx.SerializeSize()
	if size == 0 {
		return 0.0, ErrIncompleteTx
	}

	return float32(fee) / float32(size), nil
}

// UTXOs returns all the unspent transaction outputs created by this tx
func (itx *Transaction) UTXOs() UTXOs {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	var utxos UTXOs
	for i, output := range itx.MsgTx.TxOut {
		utxos = append(utxos, bitcoin.UTXO{
			Hash:          itx.Hash,
			Index:         uint32(i),
			Value:         output.Value,
			LockingScript: output.LockingScript,
		})
	}

	return utxos
}

func (itx *Transaction) IsRelevant(lockingScript bitcoin.Script) bool {
	for _, input := range itx.Inputs {
		if input.LockingScript.Equal(lockingScript) {
			return true
		}
	}
	for _, output := range itx.MsgTx.TxOut {
		if output.LockingScript.Equal(lockingScript) {
			return true
		}
	}
	return false
}

// LockingScripts returns the unique locking scripts involved in a transaction.
func (itx *Transaction) LockingScripts() []bitcoin.Script {
	result := make([]bitcoin.Script, 0, len(itx.Inputs)+len(itx.Outputs))

	for _, input := range itx.Inputs {
		if !bitcoin.LockingScriptIsUnspendable(input.LockingScript) {
			result = appendIfDoesntExist(result, input.LockingScript)
		}
	}

	for _, output := range itx.MsgTx.TxOut {
		if !bitcoin.LockingScriptIsUnspendable(output.LockingScript) {
			result = appendIfDoesntExist(result, output.LockingScript)
		}
	}

	return result
}

func appendIfDoesntExist(lockingScripts []bitcoin.Script,
	lockingScript bitcoin.Script) []bitcoin.Script {

	for _, script := range lockingScripts {
		if script.Equal(lockingScript) {
			return lockingScripts
		}
	}

	return append(lockingScripts, lockingScript)
}

func (itx *Transaction) Write(w io.Writer) error {
	// Version
	if _, err := w.Write([]byte{3}); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := itx.MsgTx.Serialize(w); err != nil {
		return errors.Wrap(err, "tx")
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(itx.Inputs))); err != nil {
		return errors.Wrap(err, "inputs count")
	}

	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Write(w); err != nil {
			return errors.Wrapf(err, "input %d", i)
		}
	}

	if _, err := w.Write([]byte{uint8(itx.RejectCode)}); err != nil {
		return errors.Wrap(err, "reject code")
	}

	return nil
}

func (itx *Transaction) Read(r io.Reader, isTest bool) error {
	// Version
	var version [1]byte
	if _, err := io.ReadFull(r, version[:]); err != nil {
		return errors.Wrap(err, "version")
	}
	if version[0] != 0 && version[0] != 1 && version[0] != 2 && version[0] != 3 {
		return fmt.Errorf("Unknown version : %d", version[0])
	}

	msg := wire.MsgTx{}
	if err := msg.Deserialize(r); err != nil {
		return errors.Wrap(err, "tx")
	}
	itx.MsgTx = &msg
	itx.Hash = *msg.TxHash()

	// Inputs
	var count uint32
	if version[0] >= 2 {
		if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
			return errors.Wrap(err, "inputs count")
		}
	} else {
		count = uint32(len(msg.TxIn))
	}

	itx.Inputs = make([]*Input, count)
	for i, _ := range itx.Inputs {
		input := &Input{}
		if err := input.Read(version[0], r); err != nil {
			return errors.Wrapf(err, "input %d", i)
		}
		itx.Inputs[i] = input
	}

	var rejectCode [1]byte
	if _, err := io.ReadFull(r, rejectCode[:]); err != nil {
		return errors.Wrap(err, "reject code")
	}
	itx.RejectCode = uint32(rejectCode[0])

	// Parse data
	for i, input := range itx.Inputs {
		action, err := protocol.Deserialize(input.LockingScript, isTest)
		if err == nil {
			itx.Inputs[i].Action = action
		}
	}

	if err := itx.ParseOutputs(isTest); err != nil {
		return errors.Wrap(err, "parse inputs")
	}

	return nil
}
