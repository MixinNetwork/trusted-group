package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
)

func (s *SQLite3Store) ListActions(ctx context.Context, state ActionState, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE action_state=? ORDER BY sequence ASC", strings.Join(actionCols, ","))
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
		a, err := actionFromRow(rows)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
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

func (s *SQLite3Store) writeAction(ctx context.Context, a *Action, ts []*Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readAction(ctx, a.OutputId)
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

		for _, t := range ts {
			exist, err := s.checkExistence(ctx, tx, "SELECT trace_id from transactions where trace_id=?", t.TraceId)
			if err != nil {
				return err
			}
			if exist {
				continue
			}
			t.State = TransactionStateInitial
			t.Sequence = a.Sequence
			err = s.execOne(ctx, tx, buildInsertionSQL("transactions", transactionCols), t.values()...)
			if err != nil {
				return fmt.Errorf("INSERT transactions %v", err)
			}
		}
	}

	return tx.Commit()
}

func (s *SQLite3Store) restoreAction(ctx context.Context, a *Action, t *Transaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.readRestorableAction(ctx, t)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("failed to find restore action of %#v %#v", a, t)
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
		a, err := actionFromRow(rows)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return as, nil
}

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, groupId, assetId string, sequence uint64, state SafeUtxoState, limit int) ([]*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE group_id=? AND asset_id=? AND sequence<=? AND state=? ORDER BY sequence ASC", strings.Join(actionCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, groupId, assetId, sequence, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var as []*Action
	for rows.Next() {
		a, err := actionFromRow(rows)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
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

	err = s.execOne(ctx, tx, "UPDATE transactions SET raw=?, hash=?, state=?, updated_at=? WHERE trace_id=?", hex.EncodeToString(t.Raw), t.Hash.String(), t.State, t.UpdatedAt, t.TraceId)
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

type Row interface {
	Scan(dest ...any) error
}
