package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

func (s *SQLite3Store) ListActions(ctx context.Context, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE state=? ORDER BY sequence ASC", strings.Join(actionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, ActionStateInitial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var as []*Action
	for rows.Next() {
		var a Action
		err := rows.Scan(&a.OutputId, &a.RequestId, &a.TransactionHash, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &a.Keys, &a.SendersThreshold, &a.Senders, &a.ReceiversThreshold, &a.Receivers, &a.Extra, &a.State, &a.Sequence, &a.CreatedAt, &a.UpdatedAt, &a.Signers, &a.SignedBy, &a.TraceId, &a.GroupId, &a.ActionState)
		if err != nil {
			return nil, err
		}
		as = append(as, &a)
	}
	return as, nil
}

func (s *SQLite3Store) readAction(ctx context.Context, id string, state ActionState) (*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE output_id=? AND action_state=?", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, id, state)
	return actionFromRow(row)
}

func (s *SQLite3Store) readRestorableAction(ctx context.Context, act *Action, ver *common.VersionedTransaction) (*Action, error) {
	if len(ver.References) != 1 {
		return nil, nil
	}
	hash := ver.References[0].String()
	query := fmt.Sprintf("SELECT %s FROM actions WHERE action_state=? AND signed_tx=? AND transaction_hash=?", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, ActionStateDone, act.TransactionHash, hash)
	return actionFromRow(row)
}

func (s *SQLite3Store) writeAction(ctx context.Context, a *Action, ts []*Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readAction(ctx, a.OutputId, a.ActionState)
	if err != nil {
		return err
	}

	if old == nil {
		if len(ts) > 0 {
			return fmt.Errorf("write new action with multiple transactions: %d", len(ts))
		}
		err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
		if err != nil {
			return fmt.Errorf("INSERT actions %v", err)
		}
	} else {
		if a.ActionState > old.ActionState {
			err = s.execOne(ctx, tx, "UPDATE actions SET action_state=? WHERE output_id=?", a.ActionState, a.OutputId)
			if err != nil {
				return fmt.Errorf("UPDATE actions %v", err)
			}
		}

		if a.ActionState == ActionStateDone {
			for _, t := range ts {
				exist, err := s.checkExistence(ctx, tx, "SELECT trace_id from transactions where trace_id=?", t.TraceId)
				if err != nil {
					return err
				}
				if exist {
					continue
				}
				t.Sequence = a.Sequence
				err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
				if err != nil {
					return fmt.Errorf("INSERT transactions %v", err)
				}
			}
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) restoreAction(ctx context.Context, a *Action, ver *common.VersionedTransaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readRestorableAction(ctx, a, ver)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("failed to find restore action of %#v %#v", a, ver)
	}

	err = s.execOne(ctx, tx, "UPDATE actions SET action_state=?, signed_by=?, signed_tx=?, restore_sequence=? WHERE output_id=?", ActionStateInitial, "", "", a.Sequence, old.OutputId)
	if err != nil {
		return fmt.Errorf("UPDATE actions %v", err)
	}

	a.ActionState = ActionStateDone
	err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
	if err != nil {
		return fmt.Errorf("INSERT actions %v", err)
	}

	return tx.Commit()
}

func (s *SQLite3Store) ListOutputsForTransaction(ctx context.Context, tx *Transaction) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE trace_id=? AND sequence<=? ORDER BY sequence ASC", strings.Join(actionCols, ","))
	rows, err := s.db.QueryContext(ctx, query, tx.TraceId, tx.Sequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var as []*Action
	for rows.Next() {
		var a Action
		err := rows.Scan(&a.OutputId, &a.RequestId, &a.TransactionHash, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &a.Keys, &a.SendersThreshold, &a.Senders, &a.ReceiversThreshold, &a.Receivers, &a.Extra, &a.State, &a.Sequence, &a.CreatedAt, &a.UpdatedAt, &a.Signers, &a.SignedBy, &a.TraceId, &a.GroupId, &a.ActionState)
		if err != nil {
			return nil, err
		}
		as = append(as, &a)
	}
	return as, nil
}

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, tx *Transaction, state SafeUtxoState, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE group_id=? AND asset_id=? AND sequence<=? AND state=? ORDER BY sequence ASC", strings.Join(actionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, tx.GroupId, tx.AssetId, tx.Sequence, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var as []*Action
	for rows.Next() {
		var a Action
		err := rows.Scan(&a.OutputId, &a.RequestId, &a.TransactionHash, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &a.Keys, &a.SendersThreshold, &a.Senders, &a.ReceiversThreshold, &a.Receivers, &a.Extra, &a.State, &a.Sequence, &a.CreatedAt, &a.UpdatedAt, &a.Signers, &a.SignedBy, &a.TraceId, &a.GroupId, &a.ActionState)
		if err != nil {
			return nil, err
		}
		as = append(as, &a)
	}
	return as, nil
}

func (s *SQLite3Store) updateTxWithOutputs(ctx context.Context, t *Transaction, os []*Action) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.execOne(ctx, tx, "UPDATE transactions SET raw=?, hash=?, state=?, updated_at=? WHERE trace_id=?", t.Raw, t.Hash, t.State, t.UpdatedAt, t.TraceId)
	if err != nil {
		return fmt.Errorf("UPDATE transactions %v", err)
	}

	for _, o := range os {
		err = s.execOne(ctx, tx, "UPDATE actions SET state=?, signed_by=?, signed_tx=? WHERE output_id=?", o.State, o.SignedBy, o.SignedTx, o.OutputId)
		if err != nil {
			return fmt.Errorf("UPDATE actions %v", err)
		}
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
		var i Iteration
		err := rows.Scan(&i.Action, &i.NodeId, &i.Threshold, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		irs = append(irs, &i)
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
	query := fmt.Sprintf("SELECT %s FROM transactions where state=? ORDER BY sequence ASC", strings.Join(iterationCols, ","))
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
		var t Transaction
		err := rows.Scan(&t.GroupId, &t.TraceId, &t.State, &t.AssetId, &t.Receivers, &t.Threshold, &t.Amount, &t.Memo, &t.Raw, &t.Hash, &t.References, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		ts = append(ts, &t)
	}
	return ts, nil
}

func (s *SQLite3Store) ReadTransactionByTraceId(ctx context.Context, id string) (*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions WHERE trace_id=?", strings.Join(transactionCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return transactionFromRow(row)
}

func (s *SQLite3Store) ReadTransactionByHash(ctx context.Context, hash crypto.Hash) (*Transaction, error) {
	query := fmt.Sprintf("SELECT %s FROM transactions WHERE hash=?", strings.Join(transactionCols, ","))
	row := s.db.QueryRowContext(ctx, query, hash.String())
	return transactionFromRow(row)
}

func (s *SQLite3Store) resetTransactionOutputs(ctx context.Context, tx *sql.Tx, traceId string) error {
	err := s.execOne(ctx, tx, "DELETE FROM outputs WHERE trace_id=?", traceId)
	if err != nil {
		return fmt.Errorf("DELETE outputs %v", err)
	}
	return err
}

func (s *SQLite3Store) DeleteTransaction(ctx context.Context, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = s.resetTransactionOutputs(ctx, tx, t.TraceId)
	if err != nil {
		return err
	}

	err = s.execOne(ctx, tx, "DELETE FROM transactions WHERE trace_id=?", t.TraceId)
	if err != nil {
		return fmt.Errorf("DELETE transactions %v", err)
	}
	return tx.Commit()
}
