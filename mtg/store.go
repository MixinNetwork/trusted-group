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
	query := fmt.Sprintf("SELECT %s FROM actions JOIN outputs ON actions.output_id = outputs.output_id WHERE action_state=? ORDER BY actions.sequence ASC", strings.Join(actionJoinCols, ","))
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

func (s *SQLite3Store) readOutput(ctx context.Context, id string) (*UnifiedOutput, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE output_id=?", strings.Join(outputCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return outputFromRow(row)
}

func (s *SQLite3Store) readAction(ctx context.Context, id string) (*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE output_id=?", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return actionFromRow(row)
}

func (s *SQLite3Store) readRestorableAction(ctx context.Context, t *Transaction) (*Action, error) {
	if len(t.References) != 1 {
		return nil, nil
	}
	hash := t.References[0].String()
	query := fmt.Sprintf("SELECT %s FROM actions WHERE action_state=? AND transaction_hash=?", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, ActionStateRrestorable, hash)
	return actionFromRow(row)
}

func (s *SQLite3Store) finishAction(ctx context.Context, id string, state ActionState, ts []*Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	act, err := s.readAction(ctx, id)
	if err != nil {
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
	gid := uuid.Must(uuid.FromString(out.GroupId))
	if gid.String() != out.GroupId {
		panic(out.GroupId)
	}

	oldAct, err := s.readAction(ctx, out.OutputId)
	if err != nil {
		return err
	}
	oldOutput, err := s.readOutput(ctx, out.OutputId)
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
	if !CheckTestEnvironment(ctx) && state != ActionStateInitial {
		panic(state)
	}
	err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
	if err != nil {
		return fmt.Errorf("INSERT actions %v", err)
	}

	return nil
}

func (s *SQLite3Store) writeAction(ctx context.Context, out *UnifiedOutput, state ActionState) error {
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

func (s *SQLite3Store) restoreAction(ctx context.Context, out *UnifiedOutput, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readAction(ctx, out.OutputId)
	if err != nil || old != nil {
		return err
	}

	act, err := s.readRestorableAction(ctx, t)
	if err != nil || act == nil {
		return err
	}

	query := "UPDATE actions SET action_state=?,restore_sequence=? WHERE output_id=? AND action_state=?"
	err = s.execOne(ctx, tx, query, ActionStateInitial, out.Sequence, act.OutputId, ActionStateRrestorable)
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
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE trace_id=? AND sequence<=? ORDER BY sequence ASC", strings.Join(outputCols, ","))
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

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, groupId, assetId string, sequence uint64, state SafeUtxoState, limit int) ([]*UnifiedOutput, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE group_id=? AND asset_id=? AND sequence<=? AND state=? ORDER BY sequence ASC", strings.Join(outputCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, groupId, assetId, sequence, state)
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

func (s *SQLite3Store) updateTxWithOutputs(ctx context.Context, t *Transaction, os []*UnifiedOutput) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.execOne(ctx, tx, "UPDATE transactions SET raw=?, hash=?, state=?, updated_at=? WHERE trace_id=?", hex.EncodeToString(t.Raw), t.Hash.String(), t.State, t.UpdatedAt, t.TraceId)
	if err != nil {
		return fmt.Errorf("UPDATE transactions %v", err)
	}

	for _, o := range os {
		err = s.execOne(ctx, tx, "UPDATE outputs SET state=?, signed_by=?, signed_tx=? WHERE output_id=?", o.State, o.SignedBy, o.SignedTx, o.OutputId)
		if err != nil {
			return fmt.Errorf("UPDATE outputs %v", err)
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) writeStorageTransaction(ctx context.Context, t *Transaction) error {
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

func (s *SQLite3Store) finishTransaction(ctx context.Context, traceId string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.execOne(ctx, tx, "UPDATE transactions SET state=?, updated_at=? WHERE trace_id=?", TransactionStateSnapshot, time.Now(), traceId)
	if err != nil {
		return fmt.Errorf("UPDATE transactions %v", err)
	}

	err = s.execOne(ctx, tx, "UPDATE actions SET state=? WHERE trace_id=?", SafeUtxoStateSpent, traceId)
	if err != nil {
		return fmt.Errorf("UPDATE actions %v", err)
	}

	return tx.Commit()
}

func (s *SQLite3Store) readIteration(ctx context.Context, id string) (*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations WHERE node_id=?", strings.Join(iterationCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return iterationFromRow(row)
}

func (s *SQLite3Store) ListIterations(ctx context.Context) ([]*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations ORDER BY node_id,created_at ASC LIMIT 255", strings.Join(iterationCols, ","))
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

	olds, err := s.ListIterations(ctx)
	if err != nil {
		return err
	}
	if len(olds) > 0 && olds[len(olds)-1].CreatedAt > ir.CreatedAt {
		panic(ir.CreatedAt)
	}

	old, err := s.readIteration(ctx, ir.NodeId)
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

func (s *SQLite3Store) ListTransactions(ctx context.Context, state, limit int) ([]*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions where state=? ORDER BY sequence ASC", strings.Join(transactionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, state)
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
