package inspector

import (
	"encoding/binary"
	"io"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

type Input struct {
	Value         uint64         `json:"value"`
	LockingScript bitcoin.Script `json:"locking_script"`

	Action actions.Action `json:"action"`
}

type Output struct {
	Action actions.Action `json:"action"`
}

// UTXOs is a wrapper for a []UTXO.
type UTXOs []bitcoin.UTXO

// Value returns the total value of the set of UTXO's.
func (utxos UTXOs) Value() uint64 {
	v := uint64(0)

	for _, utxo := range utxos {
		v += utxo.Value
	}

	return v
}

// ForLockingScript returns UTXOs that match the given locking script.
func (utxos UTXOs) ForLockingScript(lockingScript bitcoin.Script) (UTXOs, error) {
	filtered := UTXOs{}

	for _, utxo := range utxos {
		if utxo.LockingScript.Equal(lockingScript) {
			filtered = append(filtered, utxo)
		}
	}

	return filtered, nil
}

func (in Input) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, in.Value); err != nil {
		return errors.Wrap(err, "write value")
	}

	if err := wire.WriteVarInt(w, 0, uint64(len(in.LockingScript))); err != nil {
		return errors.Wrap(err, "write script length")
	}

	if _, err := w.Write(in.LockingScript); err != nil {
		return errors.Wrap(err, "write script")
	}

	return nil
}

func (in *Input) Read(version uint8, r io.Reader) error {
	switch version {
	case 0:
		// Read full tx
		msg := wire.MsgTx{}
		if err := msg.Deserialize(r); err != nil {
			return errors.Wrap(err, "read tx")
		}

	case 1, 2:
		utxo := bitcoin.UTXO{}
		if err := utxo.Read(r); err != nil {
			return errors.Wrap(err, "read utxo")
		}

		in.Value = utxo.Value
		in.LockingScript = utxo.LockingScript

	default:
		if err := binary.Read(r, binary.LittleEndian, &in.Value); err != nil {
			return errors.Wrap(err, "read value")
		}

		length, err := wire.ReadVarInt(r, 0)
		if err != nil {
			return errors.Wrap(err, "read script length")
		}

		in.LockingScript = make(bitcoin.Script, length)
		if _, err := io.ReadFull(r, in.LockingScript); err != nil {
			return errors.Wrap(err, "read script")
		}
	}

	return nil
}
