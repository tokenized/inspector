package inspector

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// func Test_Transaction_Serialize(t *testing.T) {
// 	ctx := context.Background()

// 	txb := txbuilder.NewTxBuilder(0.5, 0.25)

// 	var previousTxID bitcoin.Hash32
// 	rand.Read(previousTxID[:])

// 	fromKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	fromLockingScript, err := fromKey.LockingScript()
// 	if err != nil {
// 		t.Fatalf("Failed to locking script : %s", err)
// 	}

// 	txb.AddInputUTXO(bitcoin.UTXO{
// 		Hash:          previousTxID,
// 		Index:         1,
// 		Value:         10000,
// 		LockingScript: fromLockingScript,
// 	})

// 	changeKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	changeAddress, err := changeKey.RawAddress()
// 	if err != nil {
// 		t.Fatalf("Failed to generate address : %s", err)
// 	}

// 	txb.SetChangeAddress(changeAddress, "")

// 	toKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	toLockingScript, err := toKey.LockingScript()
// 	if err != nil {
// 		t.Fatalf("Failed to locking script : %s", err)
// 	}

// 	if err := txb.AddOutput(toLockingScript, 9000, false, false); err != nil {
// 		t.Fatalf("Failed to add output : %s", err)
// 	}

// 	offer := &actions.ContractOffer{
// 		ContractName: "Test Contract",
// 	}

// 	script, err := protocol.Serialize(offer, true)
// 	if err != nil {
// 		t.Fatalf("Failed to serialize action : %s", err)
// 	}

// 	if err := txb.AddOutput(script, 0, false, false); err != nil {
// 		t.Fatalf("Failed to add output : %s", err)
// 	}

// 	if _, err := txb.Sign([]bitcoin.Key{fromKey}); err != nil {
// 		t.Fatalf("Failed to sign tx : %s", err)
// 	}

// 	tx, err := NewTransactionFromTxBuilder(ctx, txb, true)
// 	if err != nil {
// 		t.Fatalf("Failed to create inspector tx : %s", err)
// 	}

// 	t.Logf("Tx %s", tx.String(bitcoin.MainNet))

// 	buf := &bytes.Buffer{}

// 	if err := tx.Write(buf); err != nil {
// 		t.Fatalf("Failed to write tx : %s", err)
// 	}

// 	readTx := &Transaction{}
// 	if err := readTx.Read(buf, true); err != nil {
// 		t.Fatalf("Failed to read tx : %s", err)
// 	}

// 	t.Logf("Read Tx %s", readTx.String(bitcoin.MainNet))

// 	if err := tx.Equal(*readTx); err != nil {
// 		t.Fatalf("Read tx not equal : %s", err)
// 	}
// }

// func Test_Transaction_Serialize_v2_to_v3(t *testing.T) {
// 	ctx := context.Background()

// 	txb := txbuilder.NewTxBuilder(0.5, 0.25)

// 	var previousTxID bitcoin.Hash32
// 	rand.Read(previousTxID[:])

// 	fromKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	fromLockingScript, err := fromKey.LockingScript()
// 	if err != nil {
// 		t.Fatalf("Failed to locking script : %s", err)
// 	}

// 	txb.AddInputUTXO(bitcoin.UTXO{
// 		Hash:          previousTxID,
// 		Index:         1,
// 		Value:         10000,
// 		LockingScript: fromLockingScript,
// 	})

// 	changeKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	changeAddress, err := changeKey.RawAddress()
// 	if err != nil {
// 		t.Fatalf("Failed to generate address : %s", err)
// 	}

// 	txb.SetChangeAddress(changeAddress, "")

// 	toKey, err := bitcoin.GenerateKey(bitcoin.MainNet)
// 	if err != nil {
// 		t.Fatalf("Failed to generate key : %s", err)
// 	}

// 	toLockingScript, err := toKey.LockingScript()
// 	if err != nil {
// 		t.Fatalf("Failed to locking script : %s", err)
// 	}

// 	if err := txb.AddOutput(toLockingScript, 9000, false, false); err != nil {
// 		t.Fatalf("Failed to add output : %s", err)
// 	}

// 	offer := &actions.ContractOffer{
// 		ContractName: "Test Contract",
// 	}

// 	script, err := protocol.Serialize(offer, true)
// 	if err != nil {
// 		t.Fatalf("Failed to serialize action : %s", err)
// 	}

// 	if err := txb.AddOutput(script, 0, false, false); err != nil {
// 		t.Fatalf("Failed to add output : %s", err)
// 	}

// 	if _, err := txb.Sign([]bitcoin.Key{fromKey}); err != nil {
// 		t.Fatalf("Failed to sign tx : %s", err)
// 	}

// 	tx, err := NewTransactionFromTxBuilder(ctx, txb, true)
// 	if err != nil {
// 		t.Fatalf("Failed to create inspector tx : %s", err)
// 	}

// 	t.Logf("Tx %s", tx.String(bitcoin.MainNet))

// 	buf := &bytes.Buffer{}

// 	if err := tx.Write_v2(buf); err != nil {
// 		t.Fatalf("Failed to write tx : %s", err)
// 	}

// 	readTx := &Transaction{}
// 	if err := readTx.Read(buf, true); err != nil {
// 		t.Fatalf("Failed to read tx : %s", err)
// 	}

// 	t.Logf("Read Tx %s", readTx.String(bitcoin.MainNet))

// 	if err := tx.Equal(*readTx); err != nil {
// 		t.Fatalf("Read tx not equal : %s", err)
// 	}
// }

func (itx Transaction) Equal(itx2 Transaction) error {
	if !itx.Hash.Equal(&itx2.Hash) {
		return fmt.Errorf("Wrong hash : got %s, want %s", itx.Hash, itx2.Hash)
	}

	if itx.RejectCode != itx2.RejectCode {
		return fmt.Errorf("Wrong RejectCode : got %d, want %d", itx.RejectCode, itx2.RejectCode)
	}

	buf := &bytes.Buffer{}
	if err := itx.MsgTx.Serialize(buf); err != nil {
		return errors.Wrap(err, "serialize tx")
	}

	buf2 := &bytes.Buffer{}
	if err := itx2.MsgTx.Serialize(buf2); err != nil {
		return errors.Wrap(err, "serialize tx2")
	}

	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		return fmt.Errorf("Wrong tx bytes : \ngot  %x\nwant %x", buf.Bytes(), buf2.Bytes())
	}

	if len(itx.Inputs) != len(itx2.Inputs) {
		return fmt.Errorf("Wrong Inputs count : got %d, want %d", len(itx.Inputs), len(itx2.Inputs))
	}

	for i, input := range itx.Inputs {
		if input.Value != itx2.Inputs[i].Value {
			return fmt.Errorf("Wrong input %d value : got %d, want %d", i, input.Value,
				itx2.Inputs[i].Value)
		}

		if !bytes.Equal(input.LockingScript, itx2.Inputs[i].LockingScript) {
			return fmt.Errorf("Wrong input %d locking script : \ngot  %s\nwant %s", i,
				input.LockingScript, itx2.Inputs[i].LockingScript)
		}

		if input.Action == nil {
			if itx2.Inputs[i].Action != nil {
				return fmt.Errorf("Left input %d missing action", i)
			}
		} else {
			if itx2.Inputs[i].Action == nil {
				return fmt.Errorf("Right input %d missing action", i)
			}

			script, err := protocol.Serialize(input.Action, true)
			if err != nil {
				return errors.Wrapf(err, "serialize input %d action", i)
			}

			script2, err := protocol.Serialize(itx2.Inputs[i].Action, true)
			if err != nil {
				return errors.Wrapf(err, "serialize input %d action2", i)
			}

			if !bytes.Equal(script, script2) {
				return fmt.Errorf("Wrong input %d action : \ngot  %x\nwant %x", i,
					script, script2)
			}
		}
	}

	for i, output := range itx.Outputs {
		if output.Action == nil {
			if itx2.Outputs[i].Action != nil {
				return fmt.Errorf("Left output %d missing action", i)
			}
		} else {
			if itx2.Outputs[i].Action == nil {
				return fmt.Errorf("Right output %d missing action", i)
			}

			script, err := protocol.Serialize(output.Action, true)
			if err != nil {
				return errors.Wrapf(err, "serialize output %d action", i)
			}

			script2, err := protocol.Serialize(itx2.Outputs[i].Action, true)
			if err != nil {
				return errors.Wrapf(err, "serialize output %d action2", i)
			}

			if !bytes.Equal(script, script2) {
				return fmt.Errorf("Wrong output %d action : \ngot  %x\nwant %x", i,
					script, script2)
			}
		}
	}

	return nil
}

func (itx *Transaction) Write_v2(w io.Writer) error {
	// Version
	if _, err := w.Write([]byte{2}); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := itx.MsgTx.Serialize(w); err != nil {
		return errors.Wrap(err, "tx")
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(itx.Inputs))); err != nil {
		return errors.Wrap(err, "inputs count")
	}

	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Write_v2(w); err != nil {
			return errors.Wrapf(err, "input %d", i)
		}
	}

	if _, err := w.Write([]byte{uint8(itx.RejectCode)}); err != nil {
		return errors.Wrap(err, "reject code")
	}

	return nil
}

func (in Input) Write_v2(w io.Writer) error {
	// UTXO with random hash and index values
	utxo := bitcoin.UTXO{
		Index:         1,
		Value:         in.Value,
		LockingScript: in.LockingScript,
	}
	rand.Read(utxo.Hash[:])

	if err := utxo.Write(w); err != nil {
		return err
	}

	return nil
}
