package datasource

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxListLimit es el tope máximo de filas que un connector devolverá
// en una query de lista. Evita LIMIT descontrolados inyectados por un
// resolver o cliente GraphQL.
const maxListLimit int64 = 1000

// PostgresConnector implements PersistedConnector and VectorConnector against PostgreSQL+pgvector.
type PostgresConnector struct {
	dsn       string
	pool      *pgxpool.Pool
	models    map[string]*ModelDef
	schema    string
	connected bool
}

// NewPostgresConnector creates a connector targeting the given DSN and schema.
func NewPostgresConnector(dsn string, schema string) *PostgresConnector {
	if schema == "" {
		schema = "public"
	}
	return &PostgresConnector{
		dsn:    dsn,
		models: make(map[string]*ModelDef),
		schema: schema,
	}
}

func (c *PostgresConnector) GetName() string { return "postgres" }

func (c *PostgresConnector) Connect(ctx context.Context) error {
	cfg, err := pgxpool.ParseConfig(c.dsn)
	if err != nil {
		return fmt.Errorf("parse dsn: %w", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = c.schema + ", public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("ping: %w", err)
	}
	c.pool = pool
	c.connected = true
	return nil
}

func (c *PostgresConnector) Disconnect() error {
	if c.pool != nil {
		c.pool.Close()
		c.pool = nil
	}
	c.connected = false
	return nil
}

func (c *PostgresConnector) Ping(ctx context.Context) error {
	if c.pool == nil {
		return fmt.Errorf("not connected")
	}
	return c.pool.Ping(ctx)
}

// RegisterModel stores a ModelDef for DDL generation and type resolution.
func (c *PostgresConnector) RegisterModel(model ModelDef) {
	m := model
	c.models[model.Collection] = &m
}

func (c *PostgresConnector) getModel(collection string) (*ModelDef, error) {
	m, ok := c.models[collection]
	if !ok {
		return nil, fmt.Errorf("model not registered for collection %q", collection)
	}
	return m, nil
}

// ── Migrate ──────────────────────────────────────────────────────────────────

func (c *PostgresConnector) Migrate(ctx context.Context, model ModelDef) error {
	if c.pool == nil {
		return fmt.Errorf("not connected")
	}
	c.RegisterModel(model)
	return c.ensureSchema(ctx, model)
}

func (c *PostgresConnector) ensureSchema(ctx context.Context, model ModelDef) error {
	var tableExists bool
	err := c.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=$1 AND table_name=$2)`,
		c.schema, model.Collection,
	).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("check table exists: %w", err)
	}

	if !tableExists {
		return c.createTable(ctx, model)
	}

	existing, err := c.getTableColumns(ctx, model.Collection)
	if err != nil {
		return err
	}
	existingSet := make(map[string]bool, len(existing))
	for _, col := range existing {
		existingSet[col] = true
	}
	for _, prop := range model.Properties {
		if !existingSet[prop.Name] {
			colType, err := MapPropertyType(prop.Type, prop.Length)
			if err != nil {
				return fmt.Errorf("map type for %s: %w", prop.Name, err)
			}
			nullable := "NOT NULL"
			if prop.Nullable {
				nullable = "NULL"
			}
			ddl := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s %s %s",
				pqQuoteIdent(c.schema), pqQuoteIdent(model.Collection),
				pqQuoteIdent(prop.Name), colType, nullable)
			if _, err := c.pool.Exec(ctx, ddl); err != nil {
				return fmt.Errorf("add column %s: %w", prop.Name, err)
			}
		}
	}
	return nil
}

func (c *PostgresConnector) createTable(ctx context.Context, model ModelDef) error {
	var parts []string
	for _, prop := range model.Properties {
		colType, err := MapPropertyType(prop.Type, prop.Length)
		if err != nil {
			return fmt.Errorf("map type for %s: %w", prop.Name, err)
		}
		nullable := "NOT NULL"
		if prop.Nullable {
			nullable = "NULL"
		}
		pk := ""
		if prop.PrimaryKey {
			pk = " PRIMARY KEY"
		}
		def := ""
		if prop.DefaultValue != nil {
			def = fmt.Sprintf(" DEFAULT %v", prop.DefaultValue)
		}
		parts = append(parts, fmt.Sprintf("%s %s%s %s%s",
			pqQuoteIdent(prop.Name), colType, pk, nullable, def))
	}
	ddl := fmt.Sprintf("CREATE TABLE %s.%s (\n  %s\n)",
		pqQuoteIdent(c.schema), pqQuoteIdent(model.Collection),
		strings.Join(parts, ",\n  "))
	_, err := c.pool.Exec(ctx, ddl)
	return err
}

func (c *PostgresConnector) getTableColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT column_name FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`,
		c.schema, table,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

// ── CRUD ─────────────────────────────────────────────────────────────────────

func (c *PostgresConnector) Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	var cols []string
	var placeholders []string
	var vals []interface{}
	i := 1
	for k, v := range data {
		cols = append(cols, pqQuoteIdent(k))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		vals = append(vals, v)
		i++
	}
	returnCols := buildReturnCols(data)
	sql := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s) RETURNING %s",
		pqQuoteIdent(c.schema), pqQuoteIdent(collection),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		returnCols)
	doc, err := c.queryRow(ctx, sql, vals...)
	if err != nil {
		return nil, c.mapErr(err)
	}
	return doc, nil
}

func (c *PostgresConnector) CreateMany(ctx context.Context, collection string, data []map[string]interface{}) ([]map[string]interface{}, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	if len(data) == 0 {
		return nil, nil
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var results []map[string]interface{}
	for _, d := range data {
		if len(d) == 0 {
			continue
		}
		var cols []string
		var placeholders []string
		var vals []interface{}
		i := 1
		for k, v := range d {
			cols = append(cols, pqQuoteIdent(k))
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			vals = append(vals, v)
			i++
		}
		returnCols := buildReturnCols(d)
		sql := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s) RETURNING %s",
			pqQuoteIdent(c.schema), pqQuoteIdent(collection),
			strings.Join(cols, ", "),
			strings.Join(placeholders, ", "),
			returnCols)
		doc, err := c.txQueryRow(ctx, tx, sql, vals...)
		if err != nil {
			return nil, c.mapErr(err)
		}
		results = append(results, doc)
	}
	return results, c.mapErr(tx.Commit(ctx))
}

func (c *PostgresConnector) FindById(ctx context.Context, collection string, id interface{}) (map[string]interface{}, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	model, err := c.getModel(collection)
	if err != nil {
		return nil, err
	}
	pk := c.getPrimaryKey(model)
	sql := fmt.Sprintf("SELECT * FROM %s.%s WHERE %s = $1 LIMIT 1",
		pqQuoteIdent(c.schema), pqQuoteIdent(collection), pqQuoteIdent(pk))
	doc, err := c.queryRow(ctx, sql, id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, c.mapErr(err)
	}
	return doc, nil
}

func (c *PostgresConnector) FindMany(ctx context.Context, collection string, query *Query) (Cursor, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	var where string
	var args []interface{}
	if query != nil && query.Filter != nil && len(query.Filter.Conditions) > 0 {
		where, args = buildWhereClause(query.Filter, 1)
		where = " WHERE " + where
	}

	order := ""
	if query != nil && len(query.Sort) > 0 {
		var parts []string
		for _, s := range query.Sort {
			dir := "ASC"
			if strings.EqualFold(s.Order, "desc") {
				dir = "DESC"
			}
			parts = append(parts, fmt.Sprintf("%s %s", pqQuoteIdent(s.Field), dir))
		}
		order = " ORDER BY " + strings.Join(parts, ", ")
	}

	limit := ""
	offset := ""
	if query != nil && query.Limit > 0 {
		l := query.Limit
		if l > maxListLimit {
			l = maxListLimit
		}
		limit = fmt.Sprintf(" LIMIT %d", l)
	}
	if query != nil && query.Offset > 0 {
		offset = fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	sql := fmt.Sprintf("SELECT * FROM %s.%s%s%s%s%s",
		pqQuoteIdent(c.schema), pqQuoteIdent(collection),
		where, order, limit, offset)

	rows, err := c.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, c.mapErr(err)
	}
	return &pgxCursor{rows: rows}, nil
}

func (c *PostgresConnector) Count(ctx context.Context, collection string, filter *Filter) (int64, error) {
	if c.pool == nil {
		return 0, fmt.Errorf("not connected")
	}
	var where string
	var args []interface{}
	if filter != nil && len(filter.Conditions) > 0 {
		where, args = buildWhereClause(filter, 1)
		where = " WHERE " + where
	}
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s%s",
		pqQuoteIdent(c.schema), pqQuoteIdent(collection), where)
	var count int64
	err := c.pool.QueryRow(ctx, sql, args...).Scan(&count)
	return count, c.mapErr(err)
}

func (c *PostgresConnector) UpdateById(ctx context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	if len(data) == 0 {
		return c.FindById(ctx, collection, id)
	}
	model, err := c.getModel(collection)
	if err != nil {
		return nil, err
	}
	pk := c.getPrimaryKey(model)
	var setClauses []string
	var vals []interface{}
	i := 1
	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", pqQuoteIdent(k), i))
		vals = append(vals, v)
		i++
	}
	vals = append(vals, id)
	sql := fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s = $%d RETURNING *",
		pqQuoteIdent(c.schema), pqQuoteIdent(collection),
		strings.Join(setClauses, ", "),
		pqQuoteIdent(pk), i)
	doc, err := c.queryRow(ctx, sql, vals...)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, c.mapErr(err)
	}
	return doc, nil
}

func (c *PostgresConnector) DeleteById(ctx context.Context, collection string, id interface{}) (int64, error) {
	if c.pool == nil {
		return 0, fmt.Errorf("not connected")
	}
	model, err := c.getModel(collection)
	if err != nil {
		return 0, err
	}
	pk := c.getPrimaryKey(model)
	tag, err := c.pool.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s.%s WHERE %s = $1",
			pqQuoteIdent(c.schema), pqQuoteIdent(collection), pqQuoteIdent(pk)),
		id,
	)
	if err != nil {
		return 0, c.mapErr(err)
	}
	return tag.RowsAffected(), nil
}

func (c *PostgresConnector) DeleteMany(ctx context.Context, collection string, filter *Filter) (int64, error) {
	if c.pool == nil {
		return 0, fmt.Errorf("not connected")
	}
	var where string
	var args []interface{}
	if filter != nil && len(filter.Conditions) > 0 {
		where, args = buildWhereClause(filter, 1)
		where = " WHERE " + where
	}
	tag, err := c.pool.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s.%s%s",
			pqQuoteIdent(c.schema), pqQuoteIdent(collection), where),
		args...,
	)
	if err != nil {
		return 0, c.mapErr(err)
	}
	return tag.RowsAffected(), nil
}

// ── Vector (pgvector) ────────────────────────────────────────────────────────

func (c *PostgresConnector) SearchSimilar(ctx context.Context, collection string, vec Vector, k int, filter *Filter) ([]SimilarityResult, error) {
	if c.pool == nil {
		return nil, fmt.Errorf("not connected")
	}
	model, err := c.getModel(collection)
	if err != nil {
		return nil, err
	}
	vecCol := c.getVectorColumn(model)
	if vecCol == "" {
		return nil, fmt.Errorf("no vector column found in model %q", collection)
	}
	pk := c.getPrimaryKey(model)

	var where string
	var args []interface{}
	argIdx := 2
	if filter != nil && len(filter.Conditions) > 0 {
		where, args = buildWhereClause(filter, argIdx)
		where = " AND " + where
	}

	vecLiteral := vectorToLiteral(vec)
	sql := fmt.Sprintf(
		"SELECT %s, %s <=> $1 AS distance FROM %s.%s WHERE true%s ORDER BY distance ASC LIMIT $%d",
		pqQuoteIdent(pk), pqQuoteIdent(vecCol),
		pqQuoteIdent(c.schema), pqQuoteIdent(collection),
		where, argIdx+len(args),
	)
	args = append([]interface{}{vecLiteral}, args...)
	args = append(args, k)

	rows, err := c.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SimilarityResult
	for rows.Next() {
		var sr SimilarityResult
		if err := rows.Scan(&sr.ID, &sr.Distance); err != nil {
			return nil, err
		}
		sr.Score = 1 - sr.Distance
		results = append(results, sr)
	}
	return results, rows.Err()
}

func (c *PostgresConnector) CreateVectorIndex(ctx context.Context, collection string, column string, dims int, metric string) error {
	if c.pool == nil {
		return fmt.Errorf("not connected")
	}
	if dims > 2000 {
		return nil // exact search only for >2000 dims
	}
	ops := "vector_cosine_ops"
	if metric == "l2" {
		ops = "vector_l2_ops"
	} else if metric == "ip" {
		ops = "vector_ip_ops"
	}
	idxName := fmt.Sprintf("idx_%s_%s_hnsw", collection, column)
	ddl := fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s.%s USING hnsw (%s %s)",
		pqQuoteIdent(idxName), pqQuoteIdent(c.schema), pqQuoteIdent(collection),
		pqQuoteIdent(column), ops,
	)
	_, err := c.pool.Exec(ctx, ddl)
	return err
}

func (c *PostgresConnector) mapErr(err error) error {
	if err == nil {
		return nil
	}
	return MapPGError(err)
}

// ── internal helpers ─────────────────────────────────────────────────────────

func (c *PostgresConnector) queryRow(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := c.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, pgx.ErrNoRows
	}
	return scanCurrentRow(rows)
}

func (c *PostgresConnector) txQueryRow(ctx context.Context, tx pgx.Tx, sql string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, pgx.ErrNoRows
	}
	return scanCurrentRow(rows)
}

func scanCurrentRow(rows pgx.Rows) (map[string]interface{}, error) {
	cols := rows.FieldDescriptions()
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	result := make(map[string]interface{}, len(cols))
	for i, col := range cols {
		result[col.Name] = vals[i]
	}
	return result, nil
}

func (c *PostgresConnector) getPrimaryKey(model *ModelDef) string {
	for _, p := range model.Properties {
		if p.PrimaryKey {
			return p.Name
		}
	}
	return "id"
}

func (c *PostgresConnector) getVectorColumn(model *ModelDef) string {
	for _, p := range model.Properties {
		if p.Type == PropVector {
			return p.Name
		}
	}
	return ""
}

// MapPropertyType maps a Go PropertyType to the SQL column type string.
func MapPropertyType(prop PropertyType, dims int) (string, error) {
	switch prop {
	case PropString:
		return "text", nil
	case PropInt, PropInt64:
		return "bigint", nil
	case PropFloat32:
		return "real", nil
	case PropFloat64:
		return "double precision", nil
	case PropBool:
		return "boolean", nil
	case PropTime:
		return "timestamptz", nil
	case PropBytes:
		return "bytea", nil
	case PropVector:
		if dims <= 0 {
			return "", fmt.Errorf("vector type requires positive dimensions, got %d", dims)
		}
		return fmt.Sprintf("vector(%d)", dims), nil
	case PropJSON:
		return "jsonb", nil
	default:
		return "", fmt.Errorf("unknown property type: %s", prop)
	}
}

func pqQuoteIdent(s string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(s, `"`, `""`))
}

func buildReturnCols(data map[string]interface{}) string {
	parts := make([]string, 0, len(data))
	for k := range data {
		parts = append(parts, pqQuoteIdent(k))
	}
	return strings.Join(parts, ", ")
}

func buildWhereClause(filter *Filter, startIdx int) (string, []interface{}) {
	if filter == nil || len(filter.Conditions) == 0 {
		return "", nil
	}
	logicalOp := "AND"
	if strings.EqualFold(filter.LogicalOp, "or") {
		logicalOp = "OR"
	}
	var parts []string
	var args []interface{}
	idx := startIdx
	for _, cond := range filter.Conditions {
		switch cond.Op {
		case "eq":
			parts = append(parts, fmt.Sprintf("%s = $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "neq":
			parts = append(parts, fmt.Sprintf("%s != $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "gt":
			parts = append(parts, fmt.Sprintf("%s > $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "gte":
			parts = append(parts, fmt.Sprintf("%s >= $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "lt":
			parts = append(parts, fmt.Sprintf("%s < $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "lte":
			parts = append(parts, fmt.Sprintf("%s <= $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		case "is_null":
			parts = append(parts, fmt.Sprintf("%s IS NULL", pqQuoteIdent(cond.Field)))
		case "like":
			parts = append(parts, fmt.Sprintf("%s LIKE $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		default:
			parts = append(parts, fmt.Sprintf("%s = $%d", pqQuoteIdent(cond.Field), idx))
			args = append(args, cond.Value)
			idx++
		}
	}
	return strings.Join(parts, " "+logicalOp+" "), args
}

func vectorToLiteral(vec Vector) string {
	parts := make([]string, len(vec.Values))
	for i, v := range vec.Values {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// pgxCursor wraps pgx.Rows and implements the Cursor interface.
type pgxCursor struct {
	rows    pgx.Rows
	pos     int
	current map[string]interface{}
	err     error
}

func (c *pgxCursor) Next(ctx context.Context) bool {
	if !c.rows.Next() {
		return false
	}
	c.current, c.err = scanCurrentRow(c.rows)
	c.pos++
	return c.err == nil
}

func (c *pgxCursor) Decode(val interface{}) error {
	if c.err != nil {
		return c.err
	}
	if m, ok := val.(*map[string]interface{}); ok {
		*m = c.current
	}
	return nil
}

func (c *pgxCursor) All(ctx context.Context, val interface{}) error {
	slicePtr, ok := val.(*[]map[string]interface{})
	if !ok {
		return fmt.Errorf("All requires *[]map[string]interface{}, got %T", val)
	}
	var result []map[string]interface{}
	for c.Next(ctx) {
		var doc map[string]interface{}
		if err := c.Decode(&doc); err != nil {
			return err
		}
		result = append(result, doc)
	}
	*slicePtr = result
	return c.Err()
}

func (c *pgxCursor) Close(ctx context.Context) error {
	c.rows.Close()
	return nil
}

func (c *pgxCursor) Err() error {
	if c.err != nil {
		return c.err
	}
	return c.rows.Err()
}

// Compile-time checks.
var _ PersistedConnector = (*PostgresConnector)(nil)
var _ VectorConnector = (*PostgresConnector)(nil)
var _ Cursor = (*pgxCursor)(nil)
