DROP TRIGGER IF EXISTS reference_item_key_immutable;

CREATE TRIGGER reference_item_key_immutable
BEFORE UPDATE ON reference_item
FOR EACH ROW
SET NEW.`key` = OLD.`key`;
