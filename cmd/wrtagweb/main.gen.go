// Code generated by "sqlbgen Job"; DO NOT EDIT.

package main

import "database/sql"

func (Job) PrimaryKey() string {
	return "id"
}

func (j Job) Values() []sql.NamedArg {
	return []sql.NamedArg{sql.Named("id", j.ID), sql.Named("status", j.Status), sql.Named("error", j.Error), sql.Named("operation", j.Operation), sql.Named("time", j.Time), sql.Named("use_mbid", j.UseMBID), sql.Named("source_path", j.SourcePath), sql.Named("dest_path", j.DestPath), sql.Named("search_result", j.SearchResult), sql.Named("research_links", j.ResearchLinks), sql.Named("confirm", j.Confirm)}
}

func (j *Job) ScanFrom(rows *sql.Rows) error {
	return rows.Scan(&j.ID, &j.Status, &j.Error, &j.Operation, &j.Time, &j.UseMBID, &j.SourcePath, &j.DestPath, &j.SearchResult, &j.ResearchLinks, &j.Confirm)
}
