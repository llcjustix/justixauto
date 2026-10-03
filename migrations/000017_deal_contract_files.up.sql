-- Uploaded contract scans (documents_files IDs) of a retail sale.
ALTER TABLE retail_deals ADD COLUMN contract_file_ids jsonb NOT NULL DEFAULT '[]';
