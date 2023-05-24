DROP INDEX tokenpool_networkname;
ALTER TABLE tokenpool DROP COLUMN published;
ALTER TABLE tokenpool DROP COLUMN network_name;
ALTER TABLE tokenpool DROP COLUMN plugin_data;
