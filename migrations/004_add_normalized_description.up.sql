-- Add normalized_description column to meals table
ALTER TABLE meals ADD COLUMN normalized_description TEXT;

-- Create index for faster lookups by user and normalized description
CREATE INDEX idx_meals_normalized_desc ON meals(user_id, normalized_description);

-- Add comment explaining the column
COMMENT ON COLUMN meals.normalized_description IS 'AI-normalized meal description for improved search and grouping';
