package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/crypto"
)

func (s *SQLite3Store) ListActions(ctx context.Context, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE state=? ORDER BY created_at ASC", strings.Join(actionCols, ","))
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

func (s *SQLite3Store) readRestorableAction(ctx context.Context, act *Action) (*Action, error) {
	// if len(act.References) != 1 {
	// 	return nil
	// }
	// signedTx := act.References[0]
	query := fmt.Sprintf("SELECT %s FROM actions WHERE output_id=? AND action_state=? AND signedTx=? AND transaction_hash", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, act.OutputId, ActionStateDone)
	return actionFromRow(row)
}

func (s *SQLite3Store) WriteAction(ctx context.Context, a *Action, ts []*Transaction) error {
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
			return fmt.Errorf("Write new action with multiple transactions: %d", len(ts))
		}
		if a.ActionState == ActionStateRrestorable {
			return fmt.Errorf("invalid action state: %d", a.State)
		}

		err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
		if err != nil {
			return fmt.Errorf("INSERT actions %v", err)
		}
	} else {
		if old.ActionState >= a.ActionState {
			return nil
		}
		if a.ActionState == ActionStateRrestorable {
			a.ActionState = ActionStateInitial
		}
		err = s.execOne(ctx, tx, "UPDATE actions SET action_state=? WHERE output_id=?", a.ActionState, a.OutputId)
		if err != nil {
			return fmt.Errorf("UPDATE actions %v", err)
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
				err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
				if err != nil {
					return fmt.Errorf("INSERT transactions %v", err)
				}
			}
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) RestoreAction(ctx context.Context, a *Action) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readRestorableAction(ctx, a)
	if err != nil {
		return err
	}
	if old == nil {
		reason := fmt.Sprintf("failed to find restore action of %#v", a)
		panic(reason)
	}

	err = s.execOne(ctx, tx, "UPDATE actions SET action_state=? WHERE output_id=?", ActionStateInitial, a.OutputId)
	if err != nil {
		return fmt.Errorf("UPDATE actions %v", err)
	}

	return tx.Commit()
}

func (s *SQLite3Store) ListOutputsForTransaction(ctx context.Context, traceId string) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE trace_id=? ORDER BY created_at ASC", strings.Join(actionCols, ","))
	rows, err := s.db.QueryContext(ctx, query, traceId)
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

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, groupId, assetId string, state SafeUtxoState, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE group_id=? AND trace_id=? AND state=? ORDER BY created_at ASC", strings.Join(actionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, groupId, assetId, state)
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

func (s *SQLite3Store) SignOutputs(ctx context.Context, os []*Action, traceId string) error {
	return nil
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
	query := fmt.Sprintf("SELECT %s FROM transactions where state=? ORDER BY created_at ASC", strings.Join(iterationCols, ","))
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

func (s *SQLite3Store) resetOldTransaction(ctx context.Context, tx *sql.Tx, t *Transaction) (*Transaction, error) {
	old, err := s.ReadTransactionByTraceId(ctx, t.TraceId)
	if err != nil || old == nil {
		return nil, err
	}
	switch {
	case old.State == t.State && old.Hash == t.Hash:
		return old, nil
	case t.State > old.State:
	case old.State == TransactionStateSigning && t.State == TransactionStateInitial:
		err := s.resetTransactionOutputs(ctx, tx, t.TraceId)
		if err != nil {
			return nil, err
		}
	case old.State > t.State:
		panic(old.TraceId)
	case old.Raw != nil && old.Hash != t.Hash:
		panic(old.Hash.String())
	}

	err = s.execOne(ctx, tx, "DELETE FROM transactions WHERE trace_id=?", t.TraceId)
	if err != nil {
		return nil, fmt.Errorf("DELETE transactions %v", err)
	}
	return nil, err
}

func (s *SQLite3Store) WriteTransaction(ctx context.Context, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.resetOldTransaction(ctx, tx, t)
	if err != nil || old != nil {
		return err
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
	if err != nil {
		return fmt.Errorf("INSERT transactions %v", err)
	}
	return tx.Commit()
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
