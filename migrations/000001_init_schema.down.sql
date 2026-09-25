DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS savings_settlements;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS savings_accounts;
DROP TABLE IF EXISTS class_transfer_requests;
DROP TABLE IF EXISTS teacher_class_assignments;
DROP TABLE IF EXISTS student_class_histories;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS academic_years;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS btree_gist;
DROP EXTENSION IF EXISTS pgcrypto;
