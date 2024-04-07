package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/crypto"
)

func (s *SQLite3Store) readOutput(ctx context.Context, id string) (*Output, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE utxo_id=?", strings.Join(outputCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return outputFromRow(row)
}

func (s *SQLite3Store) ListActions(ctx context.Context, limit int) ([]*UnifiedOutput, error) {
	query := "SELECT id FROM actions WHERE state=? ORDER BY created_at ASC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, ActionStateInitial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var os []*UnifiedOutput
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		o, err := s.readOutput(ctx, id)
		if err != nil {
			return nil, err
		}
		os = append(os, o.Unified())
	}
	return os, nil
}

func (s *SQLite3Store) readAction(ctx context.Context, id string) (*Action, error) {
	query := fmt.Sprintf("SELECT %s FROM actions WHERE utxo_id=?", strings.Join(actionCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return actionFromRow(row)
}

func (s *SQLite3Store) resetOldAction(ctx context.Context, tx *sql.Tx, act *Action) (*Action, error) {
	old, err := s.readAction(ctx, act.UTXOID)
	if err != nil || old == nil {
		return old, err
	}
	if old.State >= act.State {
		return old, err
	}
	err = s.execOne(ctx, tx, "DELETE FROM actions WHERE utxo_id=?", act.UTXOID)
	if err != nil {
		return nil, fmt.Errorf("DELETE actions %v", err)
	}
	return nil, err
}

func (s *SQLite3Store) WriteAction(ctx context.Context, a *Action) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	old, err := s.resetOldAction(ctx, tx, a)
	if err != nil || old != nil {
		return err
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("actions", actionCols), a.values()...)
	if err != nil {
		return fmt.Errorf("INSERT actions %v", err)
	}
	return tx.Commit()
}

func (s *SQLite3Store) resetOldOutput(ctx context.Context, tx *sql.Tx, o *Output) (*Output, error) {
	old, err := s.readOutput(ctx, o.UTXOID)
	if err != nil || old == nil {
		return old, err
	}
	switch {
	case old.State == OutputStateSigned && o.State == OutputStateUnspent:
	case o.State == OutputStateSpent && o.State > old.State:
	case old.State == o.State && old.SignedTx == o.SignedTx:
		return old, nil
	case old.State == o.State && old.SignedTx != o.SignedTx:
	case old.State > o.State:
		panic(old.UTXOID)
	case old.SignedBy != "" && old.SignedBy != o.SignedBy:
		panic(old.SignedBy)
	}

	err = s.execOne(ctx, tx, "DELETE FROM outputs WHERE utxo_id=?", o.UTXOID)
	if err != nil {
		return nil, fmt.Errorf("DELETE outputs %v", err)
	}
	return nil, err
}

func (s *SQLite3Store) ListOutputsForTransaction(ctx context.Context, traceId string) ([]*Output, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE trace_id=? ORDER BY created_at ASC", strings.Join(outputCols, ","))
	rows, err := s.db.QueryContext(ctx, query, traceId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var os []*Output
	for rows.Next() {
		var o Output
		err := rows.Scan(&o.GroupId, &o.UTXOID, &o.TraceID, &o.AssetID, &o.TransactionHash, &o.OutputIndex, &o.Amount, &o.SendersThreshold, &o.Senders, &o.Threshold, &o.Members, &o.Memo, &o.State, &o.Sequence, &o.CreatedAt, &o.UpdatedAt, &o.SignedBy, &o.SignedTx)
		if err != nil {
			return nil, err
		}
		os = append(os, &o)
	}
	return os, nil
}

func (s *SQLite3Store) ListOutputsForAsset(ctx context.Context, groupId, assetId string, state, limit int) ([]*Output, error) {
	query := fmt.Sprintf("SELECT %s FROM outputs WHERE group_id=? AND trace_id=? AND state=? ORDER BY created_at ASC", strings.Join(outputCols, ","))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.QueryContext(ctx, query, groupId, assetId, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var os []*Output
	for rows.Next() {
		var o Output
		err := rows.Scan(&o.GroupId, &o.UTXOID, &o.TraceID, &o.AssetID, &o.TransactionHash, &o.OutputIndex, &o.Amount, &o.SendersThreshold, &o.Senders, &o.Threshold, &o.Members, &o.Memo, &o.State, &o.Sequence, &o.CreatedAt, &o.UpdatedAt, &o.SignedBy, &o.SignedTx)
		if err != nil {
			return nil, err
		}
		os = append(os, &o)
	}
	return os, nil
}

// FIXME trace id
func (s *SQLite3Store) writeOutput(ctx context.Context, tx *sql.Tx, o *Output, traceId string) error {
	old, err := s.resetOldOutput(ctx, tx, o)
	if err != nil || old != nil {
		return err
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("outputs", outputCols), o.values()...)
	if err != nil {
		return fmt.Errorf("INSERT outputs %v", err)
	}
	return tx.Commit()
}

func (s *SQLite3Store) WriteOutput(ctx context.Context, o *Output, traceId string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	return s.writeOutput(ctx, tx, o, traceId)
}

func (s *SQLite3Store) WriteOutputs(ctx context.Context, os []*Output, traceId string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, o := range os {
		err = s.writeOutput(ctx, tx, o, traceId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite3Store) readIteration(ctx context.Context, id string) (*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations WHERE node_id=?", strings.Join(iterationCols, ","))
	row := s.db.QueryRowContext(ctx, query, id)
	return iterationFromRow(row)
}

func (s *SQLite3Store) ListIterations(ctx context.Context) ([]*Iteration, error) {
	query := fmt.Sprintf("SELECT %s FROM iterations ORDER BY created_at ASC LIMIT 255", strings.Join(iterationCols, ","))
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
		err = s.execOne(ctx, tx, "DELETE FROM iterations WHERE node_id=?", ir.NodeId)
		if err != nil {
			return fmt.Errorf("DELETE iterations %v", err)
		}
	}

	err = s.execOne(ctx, tx, buildInsertionSQL("iterations", iterationCols), ir.values()...)
	if err != nil {
		return fmt.Errorf("INSERT iterations %v", err)
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
