package diffutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stripe/pg-schema-diff/pkg/diff"
)

const StatementEndMarker = "-- END STATEMENT --"

func GetDDLsFromFiles(filePaths []string) ([]string, error) {
	var ddls []string
	for _, path := range filePaths {
		if strings.ToLower(filepath.Ext(path)) != ".sql" {
			return nil, fmt.Errorf("file %q is not a .sql file", path)
		}
		fileContents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file %q: %w", path, err)
		}
		// In the future, it would make sense to split the file contents into individual DDL statements; however,
		// that would require fully parsing the SQL. Naively splitting on `;` would not work because `;` can be
		// used in comments, strings, and escaped identifiers.
		ddls = append(ddls, string(fileContents))
	}
	return ddls, nil
}

func PlanToPrettyS(plan diff.Plan) string {
	sb := strings.Builder{}

	if len(plan.Statements) == 0 {
		sb.WriteString("Schema matches expected. No plan generated")
		return sb.String()
	}

	var stmtStrs []string
	for _, stmt := range plan.Statements {
		stmt = adaptStatement(stmt)
		stmtStr := statementToPrettyS(stmt)
		stmtStrs = append(stmtStrs, stmtStr)
	}
	sb.WriteString(strings.Join(stmtStrs, "\n"+StatementEndMarker+"\n\n"))
	sb.WriteString("\n")

	return sb.String()
}

func adaptStatement(stmt diff.Statement) diff.Statement {
	for _, prefix := range []string{"CREATE INDEX", "CREATE UNIQUE INDEX"} {
		concurrentPrefix := prefix + " CONCURRENTLY"
		if strings.HasPrefix(stmt.DDL, concurrentPrefix) {
			stmt.DDL = strings.Replace(stmt.DDL, concurrentPrefix, prefix, 1)
			break
		}
	}
	return stmt
}

func statementToPrettyS(stmt diff.Statement) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s;", stmt.DDL))
	if len(stmt.Hazards) > 0 {
		for _, hazard := range stmt.Hazards {
			sb.WriteString(fmt.Sprintf("\n-- [HAZARD]: %s", hazardToPrettyS(hazard)))
		}
	}
	return sb.String()
}

func hazardToPrettyS(hazard diff.MigrationHazard) string {
	if len(hazard.Message) > 0 {
		return fmt.Sprintf("%s: %s", hazard.Type, hazard.Message)
	} else {
		return hazard.Type
	}
}
