-- Migration: 0006_add_evaluations DOWN
-- Drop in reverse FK order: answer → attempt → question_option → question → evaluation

DROP TABLE IF EXISTS answer;
DROP TABLE IF EXISTS attempt;
DROP TABLE IF EXISTS question_option;
DROP TABLE IF EXISTS question;
DROP TABLE IF EXISTS evaluation;
