package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type PlaceholderFormat interface {
	ReplacePlaceholders(sql string) (string, error)
}

type dollarFormat struct{}

func (dollarFormat) ReplacePlaceholders(sql string) (string, error) {
	return replacePositionalPlaceholders(sql, "$")
}

func replacePositionalPlaceholders(sql, prefix string) (string, error) {
	buf := &bytes.Buffer{}
	i := 0
	for {
		p := strings.Index(sql, "?")
		if p == -1 {
			break
		}

		if len(sql[p:]) > 1 && sql[p:p+2] == "??" { // escape ?? => ?
			buf.WriteString(sql[:p])
			buf.WriteString("?")
			if len(sql[p:]) == 1 {
				break
			}
			sql = sql[p+2:]
		} else {
			i++
			buf.WriteString(sql[:p])
			fmt.Fprintf(buf, "%s%d", prefix, i)
			sql = sql[p+1:]
		}
	}

	buf.WriteString(sql)
	return buf.String(), nil
}

type Sqlizer interface {
	ToSql() (string, []interface{}, error)
}

type rawSqlizer interface {
	toSqlRaw() (string, []interface{}, error)
}

type part struct {
	pred interface{}
	args []interface{}
}

func newPart(pred interface{}, args ...interface{}) Sqlizer {
	return &part{pred, args}
}

func (p part) ToSql() (sql string, args []interface{}, err error) {
	switch pred := p.pred.(type) {
	case nil:
		// no-op
	case Sqlizer:
		sql, args, err = nestedToSql(pred)
	case string:
		sql = pred
		args = p.args
	default:
		err = fmt.Errorf("expected string or Sqlizer, not %T", pred)
	}
	return
}

type selectData struct {
	PlaceholderFormat PlaceholderFormat
	Options           []string
	Columns           []Sqlizer
	From              Sqlizer
	Joins             []Sqlizer
	WhereParts        []Sqlizer
	GroupBys          []string
	HavingParts       []Sqlizer
	OrderByParts      []Sqlizer
	Limit             string
	Offset            string
}

func nestedToSql(s Sqlizer) (string, []interface{}, error) {
	if raw, ok := s.(rawSqlizer); ok {
		return raw.toSqlRaw()
	} else {
		return s.ToSql()
	}
}

func appendToSql(parts []Sqlizer, w io.Writer, sep string, args []interface{}) ([]interface{}, error) {
	for i, p := range parts {
		partSql, partArgs, err := nestedToSql(p)
		if err != nil {
			return nil, err
		} else if len(partSql) == 0 {
			continue
		}

		if i > 0 {
			_, err := io.WriteString(w, sep)
			if err != nil {
				return nil, err
			}
		}

		_, err = io.WriteString(w, partSql)
		if err != nil {
			return nil, err
		}
		args = append(args, partArgs...)
	}
	return args, nil
}

func (d *selectData) ToSql() (sqlStr string, args []interface{}, err error) {
	sqlStr, args, err = d.toSqlRaw()
	if err != nil {
		return
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sqlStr)
	return
}

func (d *selectData) toSqlRaw() (sqlStr string, args []interface{}, err error) {
	if len(d.Columns) == 0 {
		err = fmt.Errorf("select statements must have at least one result column")
		return
	}

	sql := &bytes.Buffer{}

	sql.WriteString("SELECT ")

	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	if len(d.Columns) > 0 {
		args, err = appendToSql(d.Columns, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if d.From != nil {
		sql.WriteString(" FROM ")
		args, err = appendToSql([]Sqlizer{d.From}, sql, "", args)
		if err != nil {
			return
		}
	}

	if len(d.Joins) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(d.Joins, sql, " ", args)
		if err != nil {
			return
		}
	}

	if len(d.WhereParts) > 0 {
		sql.WriteString(" WHERE ")
		args, err = appendToSql(d.WhereParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(d.GroupBys) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(d.GroupBys, ", "))
	}

	if len(d.HavingParts) > 0 {
		sql.WriteString(" HAVING ")
		args, err = appendToSql(d.HavingParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(d.OrderByParts) > 0 {
		sql.WriteString(" ORDER BY ")
		args, err = appendToSql(d.OrderByParts, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if len(d.Limit) > 0 {
		sql.WriteString(" LIMIT ")
		sql.WriteString(d.Limit)
	}

	if len(d.Offset) > 0 {
		sql.WriteString(" OFFSET ")
		sql.WriteString(d.Offset)
	}

	sqlStr = sql.String()
	return
}

func main() {
	schema := `
		type User {
			id: ID!
			name: String!
			age: Int!
			orders: [Order!]
		}
		type Order {
			id: ID!
			total: Float!
		}
		type Query {
			users(age: Int, name: String): [User!]!
		}
	`

	query := `
		query {
			users(age: 30, name: "John") {
				id
				name
				orders {
					id
					total
				}
			}
		}
	`

	parsedSchema, err := gqlparser.LoadSchema(&ast.Source{
		Name:  "Schema",
		Input: schema,
	})
	if err != nil {
		fmt.Println("Error parsing schema:", err)
		return
	}

	parsedQuery, err := gqlparser.LoadQuery(parsedSchema, query)
	if err.Error() != "" {
		fmt.Println("Error parsing query:", err)
		return
	}

	for _, op := range parsedQuery.Operations {
		for _, sel := range op.SelectionSet {
			if field, ok := sel.(*ast.Field); ok {

				table := field.Name

				columns := []Sqlizer{}
				for _, sel := range field.SelectionSet {
					if field, ok := sel.(*ast.Field); ok {
						columns = append(columns, newPart(field.Name))
					}
				}

				where := []Sqlizer{}
				for _, arg := range field.Arguments {
					where = append(where, newPart(fmt.Sprintf("%s = ?", arg.Name), arg.Value))
				}

				dt := &selectData{
					PlaceholderFormat: dollarFormat{},
					Columns:           columns,
					From:              newPart(table),
					WhereParts:        where,
				}

				sql, args, err := dt.ToSql()
				if err != nil {
					panic(err)
				}
				fmt.Println(sql)
				fmt.Print(args)
			}
		}
	}
}
