CREATE TYPE snmp_version_t AS ENUM (
    '1',
    '2c',
    '3'
);

CREATE TABLE agents (
    id serial PRIMARY KEY,
    ip_address inet NOT NULL,
    port integer DEFAULT 8000,
    active boolean DEFAULT false,
    is_alive boolean DEFAULT false,
    load real DEFAULT 0,
    last_checked_at timestamp with time zone,
    UNIQUE (ip_address, port)
);

CREATE TABLE profiles (
    id serial PRIMARY KEY,
    category character varying NOT NULL,
    vendor character varying NOT NULL,
    model character varying NOT NULL,
    honor_running_only boolean DEFAULT false,
    UNIQUE(category, vendor, model)
);

COMMENT ON COLUMN profiles.honor_running_only IS 'do we honor the metric''s running_if_only flag?';

CREATE TABLE devices (
    id serial PRIMARY KEY,
    profile_id integer NOT NULL REFERENCES profiles(id),
    active boolean DEFAULT true,
    hostname character varying NOT NULL,
    ip_address character varying NOT NULL UNIQUE,
    snmp_port integer DEFAULT 161,
    snmp_version character varying DEFAULT '2c',
    snmp_community character varying NOT NULL,
    polling_frequency integer DEFAULT 300,
    is_polling boolean DEFAULT false NOT NULL,
    last_polled_at timestamp with time zone,
    snmp_timeout integer DEFAULT 10,
    snmp_retries integer DEFAULT 1,
    snmp_disable_bulk boolean DEFAULT false NOT NULL,
    snmp_connection_count integer DEFAULT 1,
    to_influx boolean DEFAULT false,
    to_kafka boolean DEFAULT true,
    to_prometheus boolean DEFAULT true,
    tags character varying DEFAULT '',
    snmpv3_security_level character varying DEFAULT '',
    snmpv3_auth_user character varying DEFAULT '',
    snmpv3_auth_passwd character varying DEFAULT '',
    snmpv3_auth_proto character varying DEFAULT '',
    snmpv3_privacy_passwd character varying DEFAULT '',
    snmpv3_privacy_proto character varying DEFAULT '',
    ping_frequency integer DEFAULT 0,
    last_pinged_at timestamp with time zone
);

CREATE TABLE metrics (
    id serial PRIMARY KEY,
    active boolean DEFAULT true,
    name character varying NOT NULL,
    oid character varying NOT NULL,
    index_pattern character varying DEFAULT '',
    description text NOT NULL,
    export_as_label boolean DEFAULT false,
    running_if_only boolean DEFAULT false,
    UNIQUE (oid, index_pattern)
);

COMMENT ON COLUMN metrics.active IS 'only active metrics are taken into account';
COMMENT ON COLUMN metrics.index_pattern IS 'regex to apply to extract index from tabular oid (only for unnatural index positions)';
COMMENT ON COLUMN metrics.export_as_label IS 'do we export this metric value as a prometheus label?';
COMMENT ON COLUMN metrics.running_if_only IS 'only query this oid on running ifaces (only active if not exported as label & profile honor_running_only flag set)';

CREATE TABLE measures (
    id serial PRIMARY KEY,
    name character varying NOT NULL,
    description text NOT NULL,
    polling_frequency integer DEFAULT 0,
    is_indexed boolean DEFAULT false,
    index_metric_id integer NOT NULL REFERENCES metrics(id),
    filter_metric_id integer NOT NULL REFERENCES metrics(id),
    filter_pattern character varying DEFAULT '',
    invert_filter_match boolean DEFAULT false
);

CREATE TABLE measure_metrics (
    id serial PRIMARY KEY,
    measure_id integer NOT NULL REFERENCES measures(id),
    metric_id integer NOT NULL REFERENCES metrics(id),
    UNIQUE (measure_id, metric_id)
);

CREATE TABLE measure_poll_times (
    id serial PRIMARY KEY,
    device_id integer NOT NULL REFERENCES devices(id),
    measure_id integer NOT NULL REFERENCES measures(id),
    last_polled_at timestamp with time zone NOT NULL,
    UNIQUE (device_id, measure_id)
);

CREATE TABLE profile_measures (
    id serial PRIMARY KEY,
    profile_id integer NOT NULL REFERENCES profiles(id),
    measure_id integer NOT NULL REFERENCES measures(id),
    UNIQUE (profile_id, measure_id)
);

CREATE TABLE reports (
    id serial PRIMARY KEY,
    uuid character varying NOT NULL UNIQUE,
    device_id integer NOT NULL REFERENCES devices(id),
    agent_id integer NOT NULL REFERENCES agents(id),
    requested_at timestamp with time zone NOT NULL,
    post_status character varying NOT NULL,
    report_received_at timestamp with time zone,
    snmp_duration_ms integer,
    snmp_error character varying DEFAULT ''
);

ALTER TYPE snmp_version_t OWNER TO horus;
ALTER TABLE agents OWNER TO horus;
ALTER TABLE profiles OWNER TO horus;
ALTER TABLE devices OWNER TO horus;
ALTER TABLE metrics OWNER TO horus;
ALTER TABLE measures OWNER TO horus;
ALTER TABLE measure_metrics OWNER TO horus;
ALTER TABLE measure_poll_times OWNER TO horus;
ALTER TABLE profile_measures OWNER TO horus;
ALTER TABLE reports OWNER TO horus;
