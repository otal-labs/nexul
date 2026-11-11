-- The secrets managers a host reports, one entry per workspace token: [{"workspace","manager","state"}], replaced whole at each report.
ALTER TABLE automation_hosts ADD COLUMN managers TEXT NOT NULL DEFAULT '[]';
