CREATE USER mattiestansbury with PASSWORD 'temp';
CREATE DATABASE foodq OWNER 'mattiestansbury';
CREATE ROLE apigateway WITH PASSWORD 'foodquser';

GRANT CONNECT ON DATABASE foodq TO apigateway;
GRANT USAGE ON SCHEMA public TO apigateway;
GRANT SELECT, UPDATE, INSERT, DELETE ON ALL TABLES IN SCHEMA public TO apigateway;
GRANT SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO apigateway;