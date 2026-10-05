-- The harness a person wrote a message in when it was relayed from there, such as 'T3'; '' for a message written in Nexul.
ALTER TABLE messages ADD COLUMN via TEXT NOT NULL DEFAULT '';
