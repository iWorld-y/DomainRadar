CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS report_runs (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS domain_reports (
    id SERIAL PRIMARY KEY,
    run_id INTEGER REFERENCES report_runs(id),
    domain_name TEXT NOT NULL,
    overview TEXT,
    trends TEXT,
    score INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS articles (
    id SERIAL PRIMARY KEY,
    domain_report_id INTEGER REFERENCES domain_reports(id),
    title TEXT,
    link TEXT,
    source TEXT,
    pub_date TEXT,
    content TEXT
);

CREATE TABLE IF NOT EXISTS key_events (
    id SERIAL PRIMARY KEY,
    domain_report_id INTEGER REFERENCES domain_reports(id),
    event_content TEXT
);

CREATE TABLE IF NOT EXISTS deep_analysis_results (
    id SERIAL PRIMARY KEY,
    run_id INTEGER REFERENCES report_runs(id),
    macro_trends TEXT,
    opportunities TEXT,
    risks TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS action_guides (
    id SERIAL PRIMARY KEY,
    deep_analysis_id INTEGER REFERENCES deep_analysis_results(id),
    guide_content TEXT
);
