-- Cria database separada pra Evolution API dentro do mesmo Postgres.
-- Rodado automaticamente na primeira vez que o container sobe (docker-entrypoint-initdb.d).
CREATE DATABASE evolution OWNER jboard;
