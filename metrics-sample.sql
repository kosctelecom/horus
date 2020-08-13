-- we define some OIDs we would like to poll
INSERT INTO metrics (id, active, name, oid, description, export_as_label, post_processors, exported_name) VALUES
(1, true, 'sysName', '.1.3.6.1.2.1.1.5.0', 'An administratively-assigned name for this managed node. By convention, it''s the node''s fqdn.', true, '{}', 'hostname'),
(2, true, 'sysUpTime', '.1.3.6.1.2.1.1.3.0', 'The system uptime in hundredths of seconds.', false, '{div:100}', 'uptime_seconds'),
(3, true, 'ifNumber', '.1.3.6.1.2.1.2.1.0', 'The number of network interfaces (regardless of their current state) present on this system.', false, '{}', 'interface_count_total'),
(4, true, 'ifIndex', ' .1.3.6.1.2.1.2.2.1.1', 'A unique index value for each interface.', false, '{}', 'index'),
(5, true, 'ifName', '.1.3.6.1.2.1.31.1.1.1.1', 'The name of the interface.', true, '{}', 'name'),
(6, true, 'ifDescr', '.1.3.6.1.2.1.2.2.1.2', 'The description of the interface.', true, '{}', 'description'),
(7, true, 'ifAdminStatus', '.1.3.6.1.2.1.2.2.1.7', 'The administrative state of the interface. Possible values: up(1), down(2), testing(3).', false, '{}', 'if_admin_status'),
(8, true, 'ifOperStatus', '.1.3.6.1.2.1.2.2.1.8', 'The operational state of the interface. Possible values: up(1), down(2), testing(3), unknown(4), dormant(5), notPresent(6).', false, '{}', 'if_oper_status'),
(9, true, 'ifHCInOctets', '.1.3.6.1.2.1.31.1.1.1.6', 'The total number of octets received on the interface, including framing characters.', false, '{parse-hex-be}', 'if_in_octets'),
(10, true, 'ifInDiscards', '.1.3.6.1.2.1.2.2.1.13', 'The number of inbound packets which were discarded.', false, '{}', 'if_in_discard_pkts'),
(11, true, 'ifInErrors', '.1.3.6.1.2.1.2.2.1.14', 'The number of inbound packets that contain errors preventing them from being deliverable.', false, '{}', 'if_in_error_pkts'),
(12, true, 'ifHCOutOctets', '.1.3.6.1.2.1.31.1.1.1.10', 'The total number of octets transmitted out of the interface, including framing characters.', false, '{parse-hex-be}', 'if_out_octets'),
(13, true, 'ifOutDiscards', '.1.3.6.1.2.1.2.2.1.19', 'The number of outbound packets which were chosen to be discarded.', false, '{}', 'if_out_discard_pkts'),
(14, true, 'ifOutErrors', '.1.3.6.1.2.1.2.2.1.20', 'The number of outbound packets that could not be transmitted because of errors.', false, '{}', 'if_out_error_pkts');

-- we define some measures to group metrics
INSERT INTO measures (id, name, description, is_indexed, index_metric_id, to_kafka, to_prometheus, to_influx, to_nats) VALUES
(1, 'sysInfo', 'basic system info', false, NULL, false, true, false, true),
(2, 'ifMetrics', 'metrics for all network interfaces', false, 4, true, true, false, true);

-- we define a generic switch profile
INSERT INTO profiles (id, category, vendor, model) VALUES
(1, 'SWITCH', 'GENERIC', 'GENERIC');

-- we associate the measures to the profiles
INSERT INTO profile_measures (profile_id, measure_id) VALUES
(1, 1),
(1, 2);

-- we associate the metrics to the measures
INSERT INTO measure_metrics (measure_id, metric_id) VALUES
(1, 1),
(1, 2),
(1, 3),
(2, 4),
(2, 5),
(2, 6),
(2, 7),
(2, 8),
(2, 9),
(2, 10),
(2, 11),
(2, 12),
(2, 13),
(2, 14);
