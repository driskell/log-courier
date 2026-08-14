/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package doris

import "testing"

func assertTypeAccepted(t *testing.T, existingType string, expectedType string) {
	t.Helper()
	mismatches := mismatchedColumns(map[string]string{"col": existingType}, map[string]string{"col": expectedType})
	if len(mismatches) != 0 {
		t.Errorf("Unexpected mismatch for existing '%s' against expected '%s'", existingType, expectedType)
	}
}

func assertTypeRejected(t *testing.T, existingType string, expectedType string) {
	t.Helper()
	mismatches := mismatchedColumns(map[string]string{"col": existingType}, map[string]string{"col": expectedType})
	if len(mismatches) == 0 {
		t.Errorf("Missing mismatch for existing '%s' against expected '%s'", existingType, expectedType)
	}
}

func TestVariantIgnoresReportedStorageProperties(t *testing.T) {
	assertTypeAccepted(t, `variant<PROPERTIES ("variant_max_subcolumns_count" = "0","variant_enable_typed_paths_to_sparse" = "false","variant_max_sparse_column_statistics_size" = "10000","variant_sparse_hash_shard_count" = "1")>`, "variant")
	assertTypeAccepted(t, "variant<>", "variant")
}

func TestVariantTypedPathsAreCompared(t *testing.T) {
	assertTypeAccepted(t, `variant<'user.id':bigint,PROPERTIES ("variant_max_subcolumns_count" = "0")>`, "variant<'user.id':bigint>")
	assertTypeRejected(t, `variant<'user.id':bigint,PROPERTIES ("variant_max_subcolumns_count" = "0")>`, "variant")
	assertTypeRejected(t, "variant<'user.id':bigint>", "variant<'user.id':int>")
	assertTypeRejected(t, "variant<'userId':int>", "variant<'userid':int>")
}

func TestStringAndTextAreInterchangeable(t *testing.T) {
	assertTypeAccepted(t, "text", "STRING")
	assertTypeAccepted(t, "string", "text")
	assertTypeAccepted(t, "TEXT", "text")
}

func TestIntegerDisplayWidthIsIgnored(t *testing.T) {
	assertTypeAccepted(t, "bigint(20)", "bigint")
	assertTypeAccepted(t, "int(11)", "INT")
	assertTypeRejected(t, "int(11)", "bigint")
}

func TestDatetimeDefaultScaleIsIgnored(t *testing.T) {
	assertTypeAccepted(t, "datetime(0)", "datetime")
	assertTypeRejected(t, "datetime(3)", "datetime")
}

func TestComplexTypeWhitespaceIsIgnored(t *testing.T) {
	assertTypeAccepted(t, "array< varchar( 255 ) >", "array<varchar(255)>")
	assertTypeAccepted(t, "ARRAY <VARCHAR(255)>", "array<varchar(255)>")
}

func TestVarcharLengthIsCompared(t *testing.T) {
	assertTypeRejected(t, "varchar(255)", "varchar(5120)")
	assertTypeRejected(t, "varchar(5120)", "varchar(255)")
}

func TestDifferentTypesAreRejected(t *testing.T) {
	assertTypeRejected(t, "varchar(255)", "bigint")
	assertTypeRejected(t, "array<varchar(255)>", "varchar(255)")
	assertTypeRejected(t, "json", "variant")
}

func TestDeployedTableSchemaIsAccepted(t *testing.T) {
	existingCols := map[string]string{
		"@timestamp": "datetime",
		"type":       "varchar(255)",
		"host":       "varchar(255)",
		"message":    "text",
		"offset":     "bigint",
		"path":       "varchar(5120)",
		"rest":       `variant<PROPERTIES ("variant_max_subcolumns_count" = "0","variant_enable_typed_paths_to_sparse" = "false","variant_max_sparse_column_statistics_size" = "10000","variant_sparse_hash_shard_count" = "1")>`,
		"tags":       "array<varchar(255)>",
		"severity":   "text",
	}
	columnDefs := map[string]string{
		"@timestamp": "datetime",
		"type":       "varchar(255)",
		"host":       "varchar(255)",
		"path":       "varchar(5120)",
		"offset":     "bigint",
		"tags":       "array<varchar(255)>",
		"message":    "text",
		"rest":       "variant",
		"severity":   "STRING",
	}

	if mismatches := mismatchedColumns(existingCols, columnDefs); len(mismatches) != 0 {
		t.Errorf("Unexpected mismatches for deployed schema: %v", mismatches)
	}
}

func TestUndefinedColumnsAreIgnored(t *testing.T) {
	existingCols := map[string]string{"col": "varchar(255)", "extra": "int"}
	columnDefs := map[string]string{"col": "varchar(255)"}

	if mismatches := mismatchedColumns(existingCols, columnDefs); len(mismatches) != 0 {
		t.Errorf("Unexpected mismatches for undefined column: %v", mismatches)
	}
}

func TestMismatchesAreReportedInNameOrder(t *testing.T) {
	existingCols := map[string]string{"zeta": "bigint", "alpha": "bigint"}
	columnDefs := map[string]string{"zeta": "varchar(255)", "alpha": "varchar(255)"}

	mismatches := mismatchedColumns(existingCols, columnDefs)
	if len(mismatches) != 2 {
		t.Fatalf("Unexpected mismatch count: got %d, want 2", len(mismatches))
	}
	if mismatches[0].name != "alpha" || mismatches[1].name != "zeta" {
		t.Errorf("Unexpected mismatch order: got %s, %s", mismatches[0].name, mismatches[1].name)
	}
}

func TestMissingColumnsAreReportedInNameOrder(t *testing.T) {
	existingCols := map[string]string{"alpha": "varchar(255)"}
	columnDefs := map[string]string{"alpha": "varchar(255)", "zeta": "bigint", "beta": "bigint"}

	missing := missingColumns(existingCols, columnDefs)
	if len(missing) != 2 || missing[0] != "beta" || missing[1] != "zeta" {
		t.Errorf("Unexpected missing columns: got %v, want [beta zeta]", missing)
	}

	if missing := missingColumns(map[string]string{"alpha": "varchar(255)"}, map[string]string{"alpha": "varchar(255)"}); len(missing) != 0 {
		t.Errorf("Unexpected missing columns for complete table: %v", missing)
	}
}
