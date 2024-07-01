package mtg

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

func (s *SQLite3Store) ListActions(ctx context.Context, state ActionState, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions JOIN outputs ON actions.output_id=outputs.output_id WHERE action_state=? ORDER BY actions.sequence ASC", strings.Join(actionJoinCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var as []*Action
	for rows.Next() {
		a, err := actionJoinFromRow(rows)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

func (s *SQLite3Store) readOutput(ctx context.Context, tx *sql.Tx, id string) (*UnifiedOutput, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE output_id=?", strings.Join(outputCols, ","))
	row := tx.QueryRowContext(ctx, query, id)
	return outputFromRow(row)
}

func (s *SQLite3Store) readAction(ctx context.Context, tx *sql.Tx, id string) (*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE output_id=?", strings.Join(actionCols, ","))
	row := tx.QueryRowContext(ctx, query, id)
	return actionFromRow(row)
}

func (s *SQLite3Store) readRestorableAction(ctx context.Context, txn *sql.Tx, t *Transaction) (*Action, error) {
	if len(t.References) != 1 {
		return nil, nil
	}
	hash := t.References[0].String()
	query := fmt.Sprintf("SELECT %s FROM actions WHERE action_state=? AND transaction_hash=?", strings.Join(actionCols, ","))
	row := txn.QueryRowContext(ctx, query, ActionStateRestorable, hash)
	return actionFromRow(row)
}

func (s *SQLite3Store) FinishAction(ctx context.Context, id string, state ActionState, ts []*Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	act, err := s.readAction(ctx, tx, id)
	if err != nil || act.ActionState != ActionStateInitial {
		return err
	}

	err = s.execOne(ctx, tx, "UPDATE actions SET action_state=? WHERE output_id=? AND action_state=?", state, id, ActionStateInitial)
	if err != nil {
		return fmt.Errorf("UPDATE actions %v", err)
	}

	for _, t := range ts {
		if t.State != TransactionStateInitial {
			panic(t.State)
		}
		sequence := act.Sequence
		if act.restoreSequence > act.Sequence {
			sequence = act.restoreSequence
		}
		if t.Sequence != sequence {
			panic(t.Sequence)
		}
		err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
		if err != nil {
			return fmt.Errorf("INSERT transactions %v", err)
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) writeOutputAndAction(ctx context.Context, tx *sql.Tx, out *UnifiedOutput, state ActionState) error {
	aid := uuid.Must(uuid.FromString(out.AppId))
	if aid.String() != out.AppId {
		panic(out.AppId)
	}

	oldAct, err := s.readAction(ctx, tx, out.OutputId)
	if err != nil {
		return err
	}
	oldOutput, err := s.readOutput(ctx, tx, out.OutputId)
	if err != nil {
		return err
	}
	switch {
	case oldAct == nil && oldOutput == nil:
	case oldAct != nil && oldOutput != nil:
		return nil
	default:
		reason := fmt.Errorf("action or output exists: %v %v", oldAct, oldOutput)
		panic(reason)
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("outputs", outputCols), out.values()...)
	if err != nil {
		return fmt.Errorf("INSERT outputs %v", err)
	}

	a := Action{
		OutputId:        out.OutputId,
		TransactionHash: out.TransactionHash.String(),
		ActionState:     state,
		Sequence:        out.Sequence,
		restoreSequence: 0,
	}
	err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
	if err != nil {
		return fmt.Errorf("INSERT actions %v", err)
	}

	return nil
}

func (s *SQLite3Store) WriteAction(ctx context.Context, out *UnifiedOutput, state ActionState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.writeOutputAndAction(ctx, tx, out, state)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLite3Store) RestoreAction(ctx context.Context, out *UnifiedOutput, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readAction(ctx, tx, out.OutputId)
	if err != nil || old != nil {
		return err
	}

	act, err := s.readRestorableAction(ctx, tx, t)
	if err != nil || act == nil {
		return err
	}

	query := "UPDATE actions SET action_state=?,restore_sequence=? WHERE output_id=? AND action_state=?"
	err = s.execOne(ctx, tx, query, ActionStateInitial, out.Sequence, act.OutputId, ActionStateRestorable)
	if err != nil {
		return fmt.Errorf("UPDATE actions %v", err)
	}

	err = s.writeOutputAndAction(ctx, tx, out, ActionStateDone)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLite3Store) ListOutputsForTransaction(ctx context.Context, traceId string, sequence uint64) ([]*UnifiedOutput, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE trace_id=? AND sequence<=? ORDER BY trace_id, sequence ASC", strings.Join(outputCols, ","))
	rows, err := s.db.QueryContext(ctx, query, traceId, sequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var os []*UnifiedOutput
	for rows.Next() {
		o, err := outputFromRow(rows)
		if err != nil {
			return nil, err
		}
		os = append(os, o)
	}
	return os, nil
}

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, appId, assetId string, sequence uint64, state SafeUtxoState, limit int) ([]*UnifiedOutput, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE app_id=? AND asset_id=? AND state=? AND sequence<=? ORDER BY app_id, asset_id, state, sequence ASC", strings.Join(outputCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, appId, assetId, state, sequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var os []*UnifiedOutput
	for rows.Next() {
		o, err := outputFromRow(rows)
		if err != nil {
			return nil, err
		}
		os = append(os, o)
	}
	return os, nil
}

func (s *SQLite3Store) UpdateTxWithOutputs(ctx context.Context, t *Transaction, os []*UnifiedOutput) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var refs []string
	for _, r := range t.References {
		refs = append(refs, r.String())
	}

	err = s.execOne(ctx, tx, "UPDATE transactions SET raw=?,hash=?,refs=?,state=?,updated_at=? WHERE trace_id=? AND state=?",
		hex.EncodeToString(t.Raw), t.Hash.String(), strings.Join(refs, ","), t.State, t.UpdatedAt, t.TraceId, TransactionStateInitial)
	if err != nil {
		return fmt.Errorf("UPDATE transactions %v", err)
	}

	for _, o := range os {
		err = s.execOne(ctx, tx, "UPDATE outputs SET trace_id=?,state=?,signed_by=?,signed_tx=? WHERE output_id=? AND state=?",
			t.TraceId, o.State, o.SignedBy, o.SignedTx, o.OutputId, SafeUtxoStateUnspent)
		if err != nil {
			return fmt.Errorf("UPDATE outputs %v", err)
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) WriteStorageTransaction(ctx context.Context, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if t.AssetId != StorageAssetId || t.Threshold != 64 || len(t.Receivers) != 1 || !t.Storage {
		return fmt.Errorf("invalid storage transaction: %#v", t)
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
	if err != nil {
		return fmt.Errorf("INSERT storage transactions %v", err)
	}

	return tx.Commit()
}

func (s *SQLite3Store) FinishTransaction(ctx context.Context, traceId string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.execOne(ctx, tx, "UPDATE transactions SET state=?, updated_at=? WHERE trace_id=? AND state=?",
		TransactionStateSnapshot, time.Now(), traceId, TransactionStateSigned)
	if err != nil {
		return fmt.Errorf("UPDATE transactions %v", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE outputs SET state=? WHERE trace_id=? AND state=?",
		SafeUtxoStateSpent, traceId, SafeUtxoStateSigned)
	if err != nil {
		return fmt.Errorf("UPDATE outputs %v", err)
	}

	return tx.Commit()
}

func (s *SQLite3Store) readIteration(ctx context.Context, txn *sql.Tx, id string) (*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations WHERE node_id=?", strings.Join(iterationCols, ","))
	row := txn.QueryRowContext(ctx, query, id)
	return iterationFromRow(row)
}

func (s *SQLite3Store) ListIterations(ctx context.Context) ([]*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations ORDER BY node_id,created_at ASC", strings.Join(iterationCols, ","))
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var irs []*Iteration
	for rows.Next() {
		i, err := iterationFromRow(rows)
		if err != nil {
			return nil, err
		}
		irs = append(irs, i)
	}
	return irs, nil
}

func (s *SQLite3Store) WriteIteration(ctx context.Context, ir *Iteration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readIteration(ctx, tx, ir.NodeId)
	if err != nil {
		return err
	}
	if old != nil && old.Action >= ir.Action {
		return nil
	}

	if old != nil {
		err = s.execOne(ctx, tx, "UPDATE iterations SET action=?, threshold=?, created_at=? WHERE node_id=?", ir.Action, ir.Threshold, ir.CreatedAt, ir.NodeId)
		if err != nil {
			return fmt.Errorf("UPDATE iterations %v", err)
		}
	} else {
		err = s.execOne(ctx, tx, buildInsertionSQL("iterations", iterationCols), ir.values()...)
		if err != nil {
			return fmt.Errorf("INSERT iterations %v", err)
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) ListPreviousInitialTransactions(ctx context.Context, asset string, sequence uint64) ([]*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions where asset_id=? AND state=? AND sequence<=? ORDER BY asset_id, state, sequence ASC", strings.Join(transactionCols, ","))
	rows, err := s.db.QueryContext(ctx, query, asset, sequence, TransactionStateInitial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ts []*Transaction
	for rows.Next() {
		t, err := transactionFromRow(rows)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func (s *SQLite3Store) ListTransactions(ctx context.Context, state, limit int) ([]*Transaction, map[string][]*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions where state=? ORDER BY state,sequence,trace_id ASC", strings.Join(transactionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, state)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var ts []*Transaction
	assetTxMap := make(map[string][]*Transaction)
	for rows.Next() {
		t, err := transactionFromRow(rows)
		if err != nil {
			return nil, nil, err
		}
		ts = append(ts, t)
		assetTxMap[t.AssetId] = append(assetTxMap[t.AssetId], t)
	}
	return ts, assetTxMap, nil
}

func (s *SQLite3Store) ReadTransactionByHash(ctx context.Context, hash crypto.Hash) (*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions WHERE hash=?", strings.Join(transactionCols, ","))
	row := s.db.QueryRowContext(ctx, query, hash.String())
	return transactionFromRow(row)
}

func (s *SQLite3Store) ReadTransactionByTraceId(ctx context.Context, id string) (*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions WHERE trace_id=?", strings.Join(transactionCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return transactionFromRow(row)
}

type Row interface {
	Scan(dest ...any) error
}
